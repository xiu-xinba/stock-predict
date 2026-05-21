using System.Globalization;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Caching.Memory;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using StockPredictApi.Clients;
using StockPredictApi.Configuration;
using StockPredictApi.Data;
using StockPredictApi.Dtos;

namespace StockPredictApi.Services;

public class PredictionService
{
    private readonly EastMoneyClient _eastMoneyClient;
    private readonly MarketService _marketService;
    private readonly OnnxPredictionService _onnxService;
    private readonly AppDbContext _db;
    private readonly IMemoryCache _cache;
    private readonly ModelSettings _settings;
    private readonly QuoteSyncService _quoteSyncService;
    private readonly SemaphoreSlim _dbWriteLock;
    private readonly ILogger<PredictionService> _logger;

    private static readonly List<FactorPoolItem> FactorPool = new()
    {
        new() { Name = "momentum_5d", Description = "5日动量因子" },
        new() { Name = "return_10d", Description = "10日收益率因子" },
        new() { Name = "volatility_20d", Description = "20日波动率因子" },
        new() { Name = "volume_ratio", Description = "成交量比率因子" },
        new() { Name = "market_beta", Description = "市场Beta因子" },
        new() { Name = "sector_momentum", Description = "行业动量因子" },
        new() { Name = "flow_signal", Description = "资金流向信号" },
        new() { Name = "mean_reversion", Description = "均值回归因子" },
    };

    public PredictionService(EastMoneyClient eastMoneyClient, MarketService marketService, OnnxPredictionService onnxService, AppDbContext db, IMemoryCache cache, IOptions<AppSettings> settings, QuoteSyncService quoteSyncService, SemaphoreSlim dbWriteLock, ILogger<PredictionService> logger)
    {
        _eastMoneyClient = eastMoneyClient;
        _marketService = marketService;
        _onnxService = onnxService;
        _db = db;
        _cache = cache;
        _settings = settings.Value.Model;
        _quoteSyncService = quoteSyncService;
        _dbWriteLock = dbWriteLock;
        _logger = logger;
    }

    public bool IsModelLoaded()
    {
        return _onnxService.IsModelLoaded();
    }

    public static Direction PctToDirection(double pct)
    {
        if (pct > 0) return Direction.Up;
        if (pct < 0) return Direction.Down;
        return Direction.Flat;
    }

    public async Task<PredictionDataDto> GenerateMockPredictionAsync(string fundCode)
    {
        var fund = await _db.Funds.FindAsync(fundCode);
        var fundName = fund?.FundName ?? $"基金{fundCode}";
        var fundType = fund?.FundType ?? "未知";

        var predictedPct = Math.Round(Random.Shared.NextDouble() * 4.0 - 2.0, 4);
        var direction = PctToDirection(predictedPct);
        var confidence = Math.Round(Random.Shared.NextDouble() * 0.3 + 0.55, 4);
        var rangeLow = Math.Round(predictedPct - Random.Shared.NextDouble() * 1.0 - 0.5, 4);
        var rangeHigh = Math.Round(predictedPct + Random.Shared.NextDouble() * 1.0 + 0.5, 4);

        var numFactors = Random.Shared.Next(3, 6);
        var selectedIndices = Enumerable.Range(0, FactorPool.Count).OrderBy(_ => Random.Shared.Next()).Take(numFactors).ToList();
        var factors = new List<FactorItemDto>();
        double totalImportance = 0;

        foreach (var idx in selectedIndices)
        {
            var imp = Math.Round(Random.Shared.NextDouble() * 0.35 + 0.05, 4);
            totalImportance += imp;
            factors.Add(new FactorItemDto
            {
                Name = FactorPool[idx].Name,
                Importance = imp,
                Description = FactorPool[idx].Description,
            });
        }

        foreach (var f in factors)
            f.Importance = Math.Round(f.Importance / totalImportance, 4);

        factors = factors.OrderByDescending(f => f.Importance).ToList();

        return new PredictionDataDto
        {
            FundCode = fundCode,
            FundName = fundName,
            Prediction = new PredictionResultDto
            {
                Direction = direction,
                DirectionConfidence = confidence,
                PredictedChangePct = predictedPct,
                ChangeRange = new ChangeRangeDto { Low = rangeLow, High = rangeHigh },
                TopFactors = factors,
            },
            MarketSnapshot = new MarketSnapshotDto
            {
                ShIndex = 3350.12,
                ShIndexChangePct = 0.0,
                SzIndex = 11000.34,
                SzIndexChangePct = 0.0,
                CybIndex = 2200.56,
                CybIndexChangePct = 0.0,
                UpdateTime = DateTime.Now.ToString("o"),
            },
        };
    }

    public async Task<ApiResponse<PredictionDataDto>> Predict(string fundCode)
    {
        if (fundCode.Length != 6 || !fundCode.All(char.IsDigit))
            return ApiResponse.Error<PredictionDataDto>(-1, "基金代码必须为6位数字");

        // Check cache BEFORE database query
        var cacheKey = $"prediction_{fundCode}";
        if (_cache.TryGetValue(cacheKey, out PredictionDataDto? cachedPrediction) && cachedPrediction != null)
            return ApiResponse.Success(cachedPrediction);

        var fund = await _db.Funds.FindAsync(fundCode);
        if (fund == null)
            return ApiResponse.Error<PredictionDataDto>(-1, $"未找到基金 {fundCode}");

        PredictionDataDto result;

        // Parallel: extract features + fetch market snapshot simultaneously
        var featuresTask = ExtractFeaturesAsync(fundCode);
        var snapshotTask = _marketService.GetIndices();
        await Task.WhenAll(featuresTask, snapshotTask);

        double[] features = await featuresTask;
        var indicesResponse = await snapshotTask;
        var onnxResult = _onnxService.Predict(features);
        if (onnxResult != null)
        {
            var (direction, confidence, probabilities) = onnxResult.Value;

            double downProb = probabilities.Length > 0 ? probabilities[0] : 0.33;
            double flatProb = probabilities.Length > 1 ? probabilities[1] : 0.34;
            double upProb = probabilities.Length > 2 ? probabilities[2] : 0.33;

            double expectedPct = Math.Round(upProb * 1.5 + flatProb * 0.0 + downProb * (-1.5), 4);

            double uncertainty = 1.0 - confidence;
            double baseSpread = 0.5 + uncertainty * 2.0;
            var rangeLow = Math.Round(expectedPct - baseSpread, 4);
            var rangeHigh = Math.Round(expectedPct + baseSpread, 4);

            var factors = _onnxService.GetTopFactors(Random.Shared.Next(3, 6));

            var hasRealFeatures = features.Any(f => f != 0);
            var reliability = hasRealFeatures ? "model" : "model_no_features";
            var reliabilityNote = hasRealFeatures
                ? "基于模型推理，特征来自实时数据近似"
                : "模型推理但特征数据缺失，结果可靠性较低";

            result = new PredictionDataDto
            {
                FundCode = fundCode,
                FundName = fund.FundName,
                Prediction = new PredictionResultDto
                {
                    Direction = direction,
                    DirectionConfidence = confidence,
                    PredictedChangePct = expectedPct,
                    ChangeRange = new ChangeRangeDto { Low = rangeLow, High = rangeHigh },
                    TopFactors = factors,
                    Reliability = reliability,
                    ReliabilityNote = reliabilityNote,
                },
                MarketSnapshot = new MarketSnapshotDto(),
            };
        }
        else
        {
            result = await GenerateMockPredictionAsync(fundCode);
            result.Prediction.Reliability = "mock";
            result.Prediction.ReliabilityNote = "模型不可用，结果为随机模拟数据，仅供参考";
        }

        // Use the already-fetched market snapshot
        if (indicesResponse.Data != null)
        {
            var idxMap = indicesResponse.Data.ToDictionary(i => i.Code);
            var sh = idxMap.GetValueOrDefault("000001");
            var sz = idxMap.GetValueOrDefault("399001");
            var cyb = idxMap.GetValueOrDefault("399006");

            result.MarketSnapshot = new MarketSnapshotDto
            {
                ShIndex = sh?.Value ?? 3350.12,
                ShIndexChangePct = sh?.ChangePct ?? 0.0,
                SzIndex = sz?.Value ?? 11000.34,
                SzIndexChangePct = sz?.ChangePct ?? 0.0,
                CybIndex = cyb?.Value ?? 2200.56,
                CybIndexChangePct = cyb?.ChangePct ?? 0.0,
                UpdateTime = DateTime.Now.ToString("o"),
            };
        }

        _cache.Set(cacheKey, result, TimeSpan.FromSeconds(_settings.CacheTtl));
        return ApiResponse.Success(result);
    }

    private async Task<double[]> ExtractFeaturesAsync(string fundCode)
    {
        var features = new double[8];
        try
        {
            var realtime = await _eastMoneyClient.FetchFundRealtime(fundCode);
            if (realtime != null)
            {
                var changePct = Convert.ToDouble(realtime["change_pct"]);
                features[0] = Math.Round(changePct * 0.8 + Random.Shared.NextDouble() * 0.4 - 0.2, 4);
                features[1] = Math.Round(changePct * 1.2 + Random.Shared.NextDouble() * 0.6 - 0.3, 4);
                features[2] = Math.Round(Math.Abs(changePct) * 0.5 + Random.Shared.NextDouble() * 0.3, 4);
                features[3] = Math.Round(Random.Shared.NextDouble() * 0.8 + 0.6, 4);
                features[4] = Math.Round(Random.Shared.NextDouble() * 0.6 + 0.7, 4);
                features[5] = Math.Round(changePct * 0.6 + Random.Shared.NextDouble() * 0.4 - 0.2, 4);
                features[6] = Math.Round(changePct * 0.4 + Random.Shared.NextDouble() * 0.3 - 0.15, 4);
                features[7] = Math.Round(-changePct * 0.3 + Random.Shared.NextDouble() * 0.2 - 0.1, 4);
            }
        }
        catch (Exception ex)
        {
            _logger.LogFeatureExtractionFailed(ex, fundCode);
        }
        return features;
    }

    public async Task<MarketSnapshotDto> GetMarketSnapshot()
    {
        try
        {
            var indicesResponse = await _marketService.GetIndices();
            if (indicesResponse.Data != null)
            {
                var idxMap = indicesResponse.Data.ToDictionary(i => i.Code);
                var sh = idxMap.GetValueOrDefault("000001");
                var sz = idxMap.GetValueOrDefault("399001");
                var cyb = idxMap.GetValueOrDefault("399006");

                return new MarketSnapshotDto
                {
                    ShIndex = sh?.Value ?? 3350.12,
                    ShIndexChangePct = sh?.ChangePct ?? 0.0,
                    SzIndex = sz?.Value ?? 11000.34,
                    SzIndexChangePct = sz?.ChangePct ?? 0.0,
                    CybIndex = cyb?.Value ?? 2200.56,
                    CybIndexChangePct = cyb?.ChangePct ?? 0.0,
                    UpdateTime = DateTime.Now.ToString("o"),
                };
            }
        }
        catch (Exception ex)
        {
            _logger.LogSnapshotFailed(ex);
        }

        return new MarketSnapshotDto
        {
            ShIndex = 3350.12,
            ShIndexChangePct = 0.0,
            SzIndex = 11000.34,
            SzIndexChangePct = 0.0,
            CybIndex = 2200.56,
            CybIndexChangePct = 0.0,
            UpdateTime = DateTime.Now.ToString("o"),
        };
    }

    public async Task<ApiResponse<List<WatchlistItemDto>>> GetWatchlistQuotes(List<string> codes)
    {
        // Top-level cache
        var cacheKey = $"watchlist_quotes_{string.Join(",", codes.Order())}";
        if (_cache.TryGetValue(cacheKey, out List<WatchlistItemDto>? cachedItems) && cachedItems != null)
            return ApiResponse.Success(cachedItems);

        var now = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
        var items = new List<WatchlistItemDto>();

        // Step 1: Read from DB first (fast, no external API call)
        var funds = await _db.Funds.Where(f => codes.Contains(f.FundCode)).ToDictionaryAsync(f => f.FundCode);

        // Step 2: Check if any fund data is stale (older than 5 minutes) or has no quote data
        var staleThreshold = DateTime.Now.AddMinutes(-5);
        var staleCodes = codes.Where(c =>
        {
            var fund = funds.GetValueOrDefault(c);
            return fund == null
                || fund.UpdatedAt == null
                || fund.UpdatedAt < staleThreshold
                || (fund.EstimatedNav == 0 && fund.ChangePct == 0 && fund.LatestNav == 0);
        }).ToList();

        // Step 3: If stale, trigger background sync and also fetch directly
        if (staleCodes.Count > 0)
        {
            try
            {
                var realtimeData = await _eastMoneyClient.FetchFundRealtimeBatch(staleCodes);
                foreach (var kvp in realtimeData)
                {
                    var fund = funds.GetValueOrDefault(kvp.Key);
                    if (fund == null) continue;

                    var data = kvp.Value;
                    var estimatedNav = Convert.ToDouble(data["estimated_nav"]);
                    var changePct = Convert.ToDouble(data["change_pct"]);
                    var latestNav = data.ContainsKey("latest_nav") ? Convert.ToDouble(data["latest_nav"]) : 0;

                    if (estimatedNav > 0)
                    {
                        fund.EstimatedNav = estimatedNav;
                        fund.ChangePct = changePct;
                    }
                    if (latestNav > 0) fund.LatestNav = latestNav;
                    fund.UpdatedAt = DateTime.Now;
                }
                await _dbWriteLock.WaitAsync();
                try
                {
                    await _db.SaveChangesAsync();
                }
                finally
                {
                    _dbWriteLock.Release();
                }
            }
            catch (Exception ex)
            {
                _logger.LogDebug(ex, "Failed to refresh stale quotes");
            }
        }

        // Step 4: Build response from DB data
        foreach (var code in codes)
        {
            var fund = funds.GetValueOrDefault(code);
            if (fund == null) continue;

            var changePct = fund.ChangePct;
            var estimatedNav = fund.EstimatedNav > 0 ? fund.EstimatedNav : fund.LatestNav;
            var direction = PctToDirection(changePct);

            items.Add(new WatchlistItemDto
            {
                FundCode = code,
                FundName = fund.FundName,
                FundType = fund.FundType,
                EstimatedNav = estimatedNav,
                ChangePct = changePct,
                Direction = direction,
                AddedAt = now,
            });
        }

        // Cache the result
        var ttl = IsTradingHours() ? TimeSpan.FromSeconds(30) : TimeSpan.FromSeconds(300);
        _cache.Set(cacheKey, items, ttl);

        // Register these funds for periodic sync by QuoteSyncService
        _quoteSyncService.RegisterActiveFunds(codes);

        return ApiResponse.Success(items);
    }

    private static bool IsTradingHours()
    {
        var now = DateTime.Now;
        var time = now.TimeOfDay;
        return time >= new TimeSpan(9, 30, 0) && time <= new TimeSpan(15, 0, 0)
            && now.DayOfWeek != DayOfWeek.Saturday && now.DayOfWeek != DayOfWeek.Sunday;
    }
}

internal sealed class FactorPoolItem
{
    public string Name { get; set; } = "";
    public string Description { get; set; } = "";
}

internal static partial class PredictionLogMessages
{
    [LoggerMessage(Level = LogLevel.Warning, Message = "Failed to fetch market indices for snapshot, using defaults")]
    public static partial void LogSnapshotFailed(this ILogger logger, Exception ex);

    [LoggerMessage(Level = LogLevel.Warning, Message = "Failed to extract features for {FundCode}, using zeros")]
    public static partial void LogFeatureExtractionFailed(this ILogger logger, Exception ex, string fundCode);
}
