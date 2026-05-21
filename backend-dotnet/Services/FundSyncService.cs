using System.Globalization;
using System.Text;
using System.Text.Json;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Caching.Memory;
using Microsoft.Extensions.Logging;
using StockPredictApi.Data;
using StockPredictApi.Models;

namespace StockPredictApi.Services;

/// <summary>
/// 基金数据同步服务：从东方财富API获取基金数据并同步到SQLite数据库
/// </summary>
public class FundSyncService : BackgroundService
{
    private readonly IServiceProvider _serviceProvider;
    private readonly IMemoryCache _cache;
    private readonly ILogger<FundSyncService> _logger;
    private readonly HttpClient _httpClient;
    private readonly SemaphoreSlim _syncLock = new(1, 1);
    private readonly SemaphoreSlim _dbWriteLock;

    private static readonly TimeSpan SyncInterval = TimeSpan.FromHours(24);
    private static readonly TimeSpan StartupDelay = TimeSpan.FromSeconds(10);

    public FundSyncService(
        IServiceProvider serviceProvider,
        IMemoryCache cache,
        SemaphoreSlim dbWriteLock,
        ILogger<FundSyncService> logger,
        IHttpClientFactory httpClientFactory)
    {
        _serviceProvider = serviceProvider;
        _cache = cache;
        _dbWriteLock = dbWriteLock;
        _logger = logger;
        _httpClient = httpClientFactory.CreateClient("FundSync");
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        await Task.Delay(StartupDelay, stoppingToken);

        // Startup check: if < 100 funds, trigger full sync
        using (var scope = _serviceProvider.CreateScope())
        {
            var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
            var fundCount = await db.Funds.CountAsync(stoppingToken);
            if (fundCount < 100)
            {
                _logger.LogInformation("Fund count {Count} < 100, triggering full sync", fundCount);
                await FullSyncAsync(stoppingToken);
            }
        }

        // Periodic incremental sync every 24 hours
        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await Task.Delay(SyncInterval, stoppingToken);
                await IncrementalSyncAsync(stoppingToken);
            }
            catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
            {
                break;
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Periodic fund sync failed");
            }
        }
    }

    /// <summary>
    /// 手动触发全量同步
    /// </summary>
    public async Task<int> FullSyncAsync(CancellationToken ct = default)
    {
        if (!await _syncLock.WaitAsync(0, ct))
            throw new InvalidOperationException("同步正在进行中，请稍后重试");

        try
        {
            using var scope = _serviceProvider.CreateScope();
            var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();

            _logger.LogInformation("Starting full fund sync from EastMoney API");
            var totalSynced = 0;
            var page = 1;
            const int pageSize = 500;

            while (!ct.IsCancellationRequested)
            {
                var funds = await FetchFundListPage(page, pageSize, ct);
                if (funds.Count == 0) break;

                foreach (var fundData in funds)
                {
                    try
                    {
                        var existing = await db.Funds.FindAsync(new object[] { fundData.FundCode }, ct);
                        if (existing != null)
                        {
                            UpdateFundFromApi(existing, fundData);
                        }
                        else
                        {
                            db.Funds.Add(fundData);
                        }
                        totalSynced++;
                    }
                    catch (Exception ex)
                    {
                        _logger.LogDebug(ex, "Failed to sync fund {Code}", fundData.FundCode);
                    }
                }

                await _dbWriteLock.WaitAsync(ct);
                try
                {
                    await db.SaveChangesAsync(ct);
                }
                finally
                {
                    _dbWriteLock.Release();
                }
                _logger.LogInformation("Synced page {Page}, total {Total} funds", page, totalSynced);

                if (funds.Count < pageSize) break;
                page++;

                // Rate limiting: 500ms between pages
                await Task.Delay(500, ct);
            }

            InvalidateSearchCache();
            _logger.LogInformation("Full sync completed, total {Total} funds", totalSynced);
            return totalSynced;
        }
        finally
        {
            _syncLock.Release();
        }
    }

    /// <summary>
    /// 增量同步：仅更新净值和收益率等时效性数据（限制最多500条）
    /// </summary>
    public async Task<int> IncrementalSyncAsync(CancellationToken ct = default)
    {
        if (!await _syncLock.WaitAsync(0, ct))
        {
            _logger.LogInformation("Sync already in progress, skipping incremental sync");
            return 0;
        }

        try
        {
            using var scope = _serviceProvider.CreateScope();
            var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();

            _logger.LogInformation("Starting incremental fund sync");
            var totalUpdated = 0;
            const int maxUpdates = 500; // Limit to prevent excessive API calls

            var funds = await db.Funds
                .OrderBy(f => f.UpdatedAt)
                .Take(maxUpdates)
                .ToListAsync(ct);

            foreach (var fund in funds)
            {
                try
                {
                    var detail = await FetchFundDetail(fund.FundCode, ct);
                    if (detail != null)
                    {
                        UpdateFundDetail(fund, detail);
                        totalUpdated++;
                    }
                }
                catch (Exception ex)
                {
                    _logger.LogDebug(ex, "Failed to update fund {Code}", fund.FundCode);
                }

                // Rate limiting
                await Task.Delay(200, ct);
            }

            await _dbWriteLock.WaitAsync(ct);
            try
            {
                await db.SaveChangesAsync(ct);
            }
            finally
            {
                _dbWriteLock.Release();
            }

            InvalidateSearchCache();
            _logger.LogInformation("Incremental sync completed, updated {Total} funds", totalUpdated);
            return totalUpdated;
        }
        finally
        {
            _syncLock.Release();
        }
    }

    private async Task<List<Fund>> FetchFundListPage(int page, int pageSize, CancellationToken ct)
    {
        var url = $"https://fund.eastmoney.com/Data/Fund_JJJZ_Data.aspx?t=1&lx=1&letter=&gsid=&text=&sort=zdf,desc&page={page},{pageSize}&dt=1530559529607&atfc=&onlySale=0";

        try
        {
            var response = await _httpClient.GetAsync(url, ct);
            response.EnsureSuccessStatusCode();
            var bytes = await response.Content.ReadAsByteArrayAsync(ct);

            // Detect encoding from response headers, fallback to GBK for Chinese sites
            var charset = response.Content.Headers.ContentType?.CharSet;
            var encoding = !string.IsNullOrEmpty(charset)
                ? Encoding.GetEncoding(charset)
                : Encoding.GetEncoding("GBK");
            var text = encoding.GetString(bytes);

            // Parse the var db = [...] format
            var startIndex = text.IndexOf("[[", StringComparison.Ordinal);
            var endIndex = text.LastIndexOf("]]", StringComparison.Ordinal);
            if (startIndex < 0 || endIndex < 0) return new List<Fund>();

            var jsonStr = text[startIndex..(endIndex + 2)];
            using var doc = JsonDocument.Parse(jsonStr);
            var root = doc.RootElement;

            var funds = new List<Fund>();
            foreach (var item in root.EnumerateArray())
            {
                if (item.ValueKind != JsonValueKind.Array) continue;
                var arr = item.EnumerateArray().ToArray();
                if (arr.Length < 5) continue;

                var fund = new Fund
                {
                    FundCode = arr[0].GetString() ?? "",
                    FundName = arr[1].GetString() ?? "",
                    FundType = GetFundTypeName(arr[3].ValueKind == JsonValueKind.Number ? arr[3].GetInt32() : 0),
                    LatestNav = arr[4].ValueKind == JsonValueKind.Number ? arr[4].GetDouble() : 0,
                    CumulativeNav = arr.Length > 5 && arr[5].ValueKind == JsonValueKind.Number ? arr[5].GetDouble() : 0,
                    UpdatedAt = DateTime.Now,
                };

                if (!string.IsNullOrEmpty(fund.FundCode) && !string.IsNullOrEmpty(fund.FundName))
                {
                    funds.Add(fund);
                }
            }

            return funds;
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Failed to fetch fund list page {Page}", page);
            return new List<Fund>();
        }
    }

    private async Task<FundDetailData?> FetchFundDetail(string fundCode, CancellationToken ct)
    {
        var url = $"https://fund.eastmoney.com/pingzhongdata/{fundCode}.js";

        try
        {
            var response = await _httpClient.GetAsync(url, ct);
            if (!response.IsSuccessStatusCode) return null;

            var bytes = await response.Content.ReadAsByteArrayAsync(ct);
            var charset = response.Content.Headers.ContentType?.CharSet;
            var encoding = !string.IsNullOrEmpty(charset)
                ? Encoding.GetEncoding(charset)
                : Encoding.GetEncoding("GBK");
            var text = encoding.GetString(bytes);

            if (string.IsNullOrEmpty(text)) return null;

            var detail = new FundDetailData();

            // Parse fS_name = "基金名称"
            var nameMatch = System.Text.RegularExpressions.Regex.Match(text, @"fS_name\s*=\s*""([^""]+)""");
            if (nameMatch.Success) detail.Name = nameMatch.Groups[1].Value;

            // Parse Data_netWorthTrend = [[date,nav,accNav,...],...]
            var navMatch = System.Text.RegularExpressions.Regex.Match(text, @"Data_netWorthTrend\s*=\s*(\[.*?\]);", System.Text.RegularExpressions.RegexOptions.Singleline);
            if (navMatch.Success)
            {
                try
                {
                    using var doc = JsonDocument.Parse(navMatch.Groups[1].Value);
                    var arr = doc.RootElement.EnumerateArray().ToArray();
                    if (arr.Length > 0)
                    {
                        var lastEntry = arr[^1];
                        if (lastEntry.ValueKind == JsonValueKind.Array)
                        {
                            var entryArr = lastEntry.EnumerateArray().ToArray();
                            if (entryArr.Length >= 2 && entryArr[1].ValueKind == JsonValueKind.Number)
                                detail.LatestNav = entryArr[1].GetDouble();
                            if (entryArr.Length >= 3 && entryArr[2].ValueKind == JsonValueKind.Number)
                                detail.CumulativeNav = entryArr[2].GetDouble();
                        }
                    }
                }
                catch
                {
                    // Ignore parse errors for nav data
                }
            }

            // Parse return rates from Data_performance
            var perfMatch = System.Text.RegularExpressions.Regex.Match(text, @"Data_performance\s*=\s*(\[.*?\]);", System.Text.RegularExpressions.RegexOptions.Singleline);
            if (perfMatch.Success)
            {
                try
                {
                    using var doc = JsonDocument.Parse(perfMatch.Groups[1].Value);
                    if (doc.RootElement.ValueKind == JsonValueKind.Array)
                    {
                        var perfArr = doc.RootElement.EnumerateArray().ToArray();
                        if (perfArr.Length > 1 && perfArr[1].ValueKind == JsonValueKind.Number)
                            detail.Return1M = perfArr[1].GetDouble();
                        if (perfArr.Length > 2 && perfArr[2].ValueKind == JsonValueKind.Number)
                            detail.Return3M = perfArr[2].GetDouble();
                        if (perfArr.Length > 3 && perfArr[3].ValueKind == JsonValueKind.Number)
                            detail.Return6M = perfArr[3].GetDouble();
                        if (perfArr.Length > 4 && perfArr[4].ValueKind == JsonValueKind.Number)
                            detail.Return1Y = perfArr[4].GetDouble();
                        if (perfArr.Length > 5 && perfArr[5].ValueKind == JsonValueKind.Number)
                            detail.Return3Y = perfArr[5].GetDouble();
                    }
                }
                catch
                {
                    // Ignore parse errors for performance data
                }
            }

            return detail;
        }
        catch (Exception ex)
        {
            _logger.LogDebug(ex, "Failed to fetch fund detail for {Code}", fundCode);
            return null;
        }
    }

    private static void UpdateFundFromApi(Fund existing, Fund newData)
    {
        existing.FundName = newData.FundName;
        existing.FundType = newData.FundType;
        // Always update nav values (0 is a valid value from API)
        existing.LatestNav = newData.LatestNav;
        existing.CumulativeNav = newData.CumulativeNav;
        existing.UpdatedAt = DateTime.Now;
    }

    private static void UpdateFundDetail(Fund fund, FundDetailData detail)
    {
        // Always update from detail data (0 is a valid return value)
        fund.LatestNav = detail.LatestNav;
        fund.CumulativeNav = detail.CumulativeNav;
        fund.Return1M = detail.Return1M;
        fund.Return3M = detail.Return3M;
        fund.Return6M = detail.Return6M;
        fund.Return1Y = detail.Return1Y;
        fund.Return3Y = detail.Return3Y;
        fund.UpdatedAt = DateTime.Now;
    }

    private void InvalidateSearchCache()
    {
        _cache.Remove("fund_filters");
        // Compact is available on MemoryCache concrete type
        if (_cache is MemoryCache mc)
            mc.Compact(1.0);
    }

    private static string GetFundTypeName(int typeCode) => typeCode switch
    {
        1 => "股票型",
        2 => "混合型",
        3 => "债券型",
        4 => "指数型",
        5 => "QDII",
        6 => "FOF",
        7 => "货币型",
        8 => "另类投资",
        _ => "其他",
    };

    private sealed class FundDetailData
    {
        public string Name { get; set; } = "";
        public double LatestNav { get; set; }
        public double CumulativeNav { get; set; }
        public double Return1M { get; set; }
        public double Return3M { get; set; }
        public double Return6M { get; set; }
        public double Return1Y { get; set; }
        public double Return3Y { get; set; }
    }
}
