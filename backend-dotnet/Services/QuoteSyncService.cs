using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging;
using StockPredictApi.Clients;
using StockPredictApi.Data;

namespace StockPredictApi.Services;

/// <summary>
/// 基金净值定时同步服务：定期从腾讯API获取用户关注的基金净值数据，写入数据库
/// 仅同步用户实际请求过的基金（通过 GetWatchlistQuotes 触发记录）
/// </summary>
public class QuoteSyncService : BackgroundService
{
    private readonly IServiceProvider _serviceProvider;
    private readonly ILogger<QuoteSyncService> _logger;
    private readonly SemaphoreSlim _dbWriteLock;

    private static readonly TimeSpan TradingInterval = TimeSpan.FromMinutes(2);
    private static readonly TimeSpan NonTradingInterval = TimeSpan.FromMinutes(30);
    private static readonly TimeSpan StartupDelay = TimeSpan.FromSeconds(15);

    // Track which fund codes have been requested by users
    private readonly HashSet<string> _activeFundCodes = new();
    private readonly object _lock = new();

    public QuoteSyncService(
        IServiceProvider serviceProvider,
        SemaphoreSlim dbWriteLock,
        ILogger<QuoteSyncService> logger)
    {
        _serviceProvider = serviceProvider;
        _dbWriteLock = dbWriteLock;
        _logger = logger;
    }

    /// <summary>
    /// 注册用户请求的基金代码，这些基金将被定时同步
    /// </summary>
    public void RegisterActiveFunds(IEnumerable<string> codes)
    {
        lock (_lock)
        {
            foreach (var code in codes)
            {
                _activeFundCodes.Add(code);
            }
        }
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        await Task.Delay(StartupDelay, stoppingToken);

        while (!stoppingToken.IsCancellationRequested)
        {
            var interval = IsTradingHours() ? TradingInterval : NonTradingInterval;
            try
            {
                await Task.Delay(interval, stoppingToken);
                await SyncActiveFundQuotes(stoppingToken);
            }
            catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
            {
                break;
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Quote sync failed");
            }
        }
    }

    private async Task SyncActiveFundQuotes(CancellationToken ct)
    {
        List<string> codesToSync;
        lock (_lock)
        {
            codesToSync = _activeFundCodes.ToList();
        }

        if (codesToSync.Count == 0) return;

        _logger.LogInformation("Starting quote sync for {Count} active funds", codesToSync.Count);

        using var scope = _serviceProvider.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
        var eastMoneyClient = scope.ServiceProvider.GetRequiredService<EastMoneyClient>();

        var totalUpdated = 0;

        foreach (var batch in codesToSync.Chunk(50))
        {
            if (ct.IsCancellationRequested) break;

            try
            {
                var batchData = await eastMoneyClient.FetchFundRealtimeBatch(batch);

                foreach (var kvp in batchData)
                {
                    var fund = await db.Funds.FindAsync(new object[] { kvp.Key }, ct);
                    if (fund == null) continue;

                    var data = kvp.Value;
                    var estimatedNav = Convert.ToDouble(data["estimated_nav"]);
                    var changePct = Convert.ToDouble(data["change_pct"]);
                    var latestNav = data.ContainsKey("latest_nav") ? Convert.ToDouble(data["latest_nav"]) : 0;
                    var cumulativeNav = data.ContainsKey("cumulative_nav") ? Convert.ToDouble(data["cumulative_nav"]) : 0;

                    if (estimatedNav > 0)
                    {
                        fund.EstimatedNav = estimatedNav;
                        fund.ChangePct = changePct;
                    }
                    if (latestNav > 0) fund.LatestNav = latestNav;
                    if (cumulativeNav > 0) fund.CumulativeNav = cumulativeNav;
                    fund.UpdatedAt = DateTime.Now;
                    totalUpdated++;
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
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "Failed to sync batch");
            }

            await Task.Delay(200, ct);
        }

        _logger.LogInformation("Quote sync completed, updated {Total} funds", totalUpdated);
    }

    /// <summary>
    /// 手动触发同步指定基金的净值
    /// </summary>
    public async Task SyncSpecificFunds(IEnumerable<string> fundCodes, CancellationToken ct = default)
    {
        using var scope = _serviceProvider.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
        var eastMoneyClient = scope.ServiceProvider.GetRequiredService<EastMoneyClient>();

        var codesList = fundCodes.ToList();
        if (codesList.Count == 0) return;

        var batchData = await eastMoneyClient.FetchFundRealtimeBatch(codesList);

        foreach (var kvp in batchData)
        {
            var fund = await db.Funds.FindAsync(new object[] { kvp.Key }, ct);
            if (fund == null) continue;

            var data = kvp.Value;
            var estimatedNav = Convert.ToDouble(data["estimated_nav"]);
            var changePct = Convert.ToDouble(data["change_pct"]);
            var latestNav = data.ContainsKey("latest_nav") ? Convert.ToDouble(data["latest_nav"]) : 0;
            var cumulativeNav = data.ContainsKey("cumulative_nav") ? Convert.ToDouble(data["cumulative_nav"]) : 0;

            if (estimatedNav > 0)
            {
                fund.EstimatedNav = estimatedNav;
                fund.ChangePct = changePct;
            }
            if (latestNav > 0) fund.LatestNav = latestNav;
            if (cumulativeNav > 0) fund.CumulativeNav = cumulativeNav;
            fund.UpdatedAt = DateTime.Now;
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
    }

    private static bool IsTradingHours()
    {
        var now = DateTime.Now;
        var time = now.TimeOfDay;
        return time >= new TimeSpan(9, 30, 0) && time <= new TimeSpan(15, 0, 0)
            && now.DayOfWeek != DayOfWeek.Saturday && now.DayOfWeek != DayOfWeek.Sunday;
    }
}
