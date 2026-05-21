using Microsoft.Extensions.Caching.Memory;
using StockPredictApi.Clients;
using StockPredictApi.Configuration;
using StockPredictApi.Data;
using StockPredictApi.Dtos;

namespace StockPredictApi.Services;

public class MarketService
{
    private readonly EastMoneyClient _eastMoneyClient;
    private readonly AppDbContext _db;
    private readonly IMemoryCache _cache;
    private readonly MarketSettings _settings;

    public MarketService(EastMoneyClient eastMoneyClient, AppDbContext db, IMemoryCache cache, Microsoft.Extensions.Options.IOptions<AppSettings> settings)
    {
        _eastMoneyClient = eastMoneyClient;
        _db = db;
        _cache = cache;
        _settings = settings.Value.Market;
    }

    public async Task<ApiResponse<List<MarketIndexDto>>> GetIndices()
    {
        var cacheKey = "market_indices";
        if (_cache.TryGetValue(cacheKey, out List<MarketIndexDto>? cached))
            return ApiResponse.Success(cached!);

        var indices = new List<MarketIndexDto>();

        // Step 1: Fetch all realtime data in parallel
        var realtimeTasks = EastMoneyClient.MarketIndicesConfig.Select(
            config => _eastMoneyClient.FetchRealtimeIndex(config.Code, config.TencentPrefix, config.Market, config.TencentCode));
        var realtimeResults = await Task.WhenAll(realtimeTasks);

        // Step 2: Fetch all chart data in parallel
        var chartTasks = new List<Task<List<double>>>();
        for (int i = 0; i < EastMoneyClient.MarketIndicesConfig.Count; i++)
        {
            var config = EastMoneyClient.MarketIndicesConfig[i];
            var realData = realtimeResults[i];
            chartTasks.Add(realData != null
                ? _eastMoneyClient.FetchIndexMinichart(config.Code, config.Market)
                : Task.FromResult(new List<double>()));
        }
        var chartResults = await Task.WhenAll(chartTasks);

        // Step 3: Build results
        for (int i = 0; i < EastMoneyClient.MarketIndicesConfig.Count; i++)
        {
            var config = EastMoneyClient.MarketIndicesConfig[i];
            var realData = realtimeResults[i];
            var chartData = chartResults[i];

            if (realData != null)
            {
                if (chartData.Count == 0)
                {
                    chartData = EastMoneyClient.GenerateIntradayChart(realData.Value, realData.ChangePct);
                }
                realData.MiniChartData = chartData;
                realData.UpdateTime = DateTime.Now.ToString("o");
                realData.DataSource = "real";
                realData.Market = config.Market;
                indices.Add(realData);
            }
            else
            {
                var simIndex = _eastMoneyClient.GenerateSimulatedIndex(config.Code);
                indices.Add(simIndex);
            }
        }

        var ttl = IsTradingHours() ? TimeSpan.FromSeconds(_settings.CacheTtl) : TimeSpan.FromSeconds(300);
        _cache.Set(cacheKey, indices, ttl);
        return ApiResponse.Success(indices);
    }

    public ApiResponse<List<FundRankingItemDto>> GetRanking(string rankingType, int size = 5)
    {
        var cacheKey = $"ranking_{rankingType}_{size}";
        if (_cache.TryGetValue(cacheKey, out List<FundRankingItemDto>? cached))
            return ApiResponse.Success(cached!);

        var query = _db.Funds.Where(f => f.Return1Y != 0);

        var funds = rankingType == "gainers"
            ? query.OrderByDescending(f => f.Return1Y).Take(size).ToList()
            : query.OrderBy(f => f.Return1Y).Take(size).ToList();

        var items = funds.Select((fund, i) => new FundRankingItemDto
        {
            Rank = i + 1,
            FundCode = fund.FundCode,
            FundName = fund.FundName,
            FundType = fund.FundType,
            ChangePct = Math.Round(fund.Return1Y, 2),
            EstimatedNav = Math.Round(fund.LatestNav, 4),
        }).ToList();

        _cache.Set(cacheKey, items, TimeSpan.FromSeconds(60));
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
