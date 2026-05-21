using System.Globalization;
using System.Text;
using System.Text.Json;
using Microsoft.Extensions.Caching.Memory;
using Microsoft.Extensions.Logging;
using StockPredictApi.Configuration;
using StockPredictApi.Dtos;

namespace StockPredictApi.Clients;

public class EastMoneyClient
{
    private readonly HttpClient _httpClient;
    private readonly EastMoneySettings _settings;
    private readonly IMemoryCache _cache;
    private readonly ILogger<EastMoneyClient> _logger;

    private static readonly string[] ShanghaiPrefixes = { "000", "600", "601", "603", "605" };

    public static readonly List<FundItemDto> BuiltinFunds = new()
    {
        new() { FundCode = "000001", FundName = "华夏成长混合", FundType = "混合型" },
        new() { FundCode = "000011", FundName = "华夏大盘精选混合", FundType = "混合型" },
        new() { FundCode = "000021", FundName = "华夏优势增长混合", FundType = "混合型" },
        new() { FundCode = "000031", FundName = "华夏复兴混合", FundType = "混合型" },
        new() { FundCode = "000041", FundName = "华夏全球精选", FundType = "QDII" },
        new() { FundCode = "000051", FundName = "华夏沪深300ETF联接", FundType = "指数型" },
        new() { FundCode = "000061", FundName = "华夏盛世精选混合", FundType = "混合型" },
        new() { FundCode = "000071", FundName = "华夏恒生ETF联接", FundType = "QDII" },
        new() { FundCode = "000091", FundName = "华夏移动互联混合", FundType = "混合型" },
        new() { FundCode = "000101", FundName = "华夏红利混合", FundType = "混合型" },
        new() { FundCode = "110011", FundName = "易方达中小盘混合", FundType = "混合型" },
        new() { FundCode = "110022", FundName = "易方达消费行业股票", FundType = "股票型" },
        new() { FundCode = "005827", FundName = "易方达蓝筹精选混合", FundType = "混合型" },
        new() { FundCode = "161725", FundName = "招商中证白酒指数", FundType = "指数型" },
        new() { FundCode = "003834", FundName = "华夏能源革新股票", FundType = "股票型" },
        new() { FundCode = "005911", FundName = "广发双擎升级混合", FundType = "混合型" },
        new() { FundCode = "007119", FundName = "景顺长城绩优成长混合", FundType = "混合型" },
        new() { FundCode = "001938", FundName = "中欧时代先锋股票", FundType = "股票型" },
        new() { FundCode = "001156", FundName = "申万菱信新能源汽车", FundType = "混合型" },
        new() { FundCode = "519736", FundName = "交银新成长混合", FundType = "混合型" },
        new() { FundCode = "001632", FundName = "天弘创新驱动混合", FundType = "混合型" },
        new() { FundCode = "001714", FundName = "工银前沿医疗股票", FundType = "股票型" },
        new() { FundCode = "001875", FundName = "前海开源公用事业股票", FundType = "股票型" },
        new() { FundCode = "002190", FundName = "农银汇理新能源主题", FundType = "混合型" },
        new() { FundCode = "002621", FundName = "中欧医疗创新股票", FundType = "股票型" },
        new() { FundCode = "003096", FundName = "中欧医疗健康混合", FundType = "混合型" },
        new() { FundCode = "004851", FundName = "广发医疗保健股票", FundType = "股票型" },
        new() { FundCode = "005818", FundName = "景顺长城新兴成长混合", FundType = "混合型" },
        new() { FundCode = "006228", FundName = "中欧时代智慧混合", FundType = "混合型" },
        new() { FundCode = "007874", FundName = "华夏科技创新混合", FundType = "混合型" },
    };

    public static readonly List<MarketIndexConfig> MarketIndicesConfig = new()
    {
        // A股
        new() { Code = "000001", Name = "上证指数", Market = "cn" },
        new() { Code = "399001", Name = "深证成指", Market = "cn" },
        new() { Code = "399006", Name = "创业板指", Market = "cn" },
        // 港股
        new() { Code = "HSI", Name = "恒生指数", Market = "hk", TencentPrefix = "hk" },
        new() { Code = "HSTECH", Name = "恒生科技指数", Market = "hk", TencentPrefix = "hk" },
        // 美股
        new() { Code = "DJI", Name = "道琼斯工业平均", Market = "us", TencentPrefix = "us" },
        new() { Code = "IXIC", Name = "纳斯达克综合", Market = "us", TencentPrefix = "us" },
        new() { Code = "SPX", Name = "标普500", Market = "us", TencentPrefix = "us", TencentCode = ".INX" },
    };

    public static readonly Dictionary<string, SimulatedIndexData> SimulatedIndexData = new()
    {
        ["000001"] = new() { Name = "上证指数", BaseValue = 3500.0, Volatility = 0.02, Market = "cn" },
        ["399001"] = new() { Name = "深证成指", BaseValue = 12000.0, Volatility = 0.025, Market = "cn" },
        ["399006"] = new() { Name = "创业板指", BaseValue = 2600.0, Volatility = 0.03, Market = "cn" },
        ["HSI"] = new() { Name = "恒生指数", BaseValue = 25000.0, Volatility = 0.02, Market = "hk" },
        ["HSTECH"] = new() { Name = "恒生科技指数", BaseValue = 5000.0, Volatility = 0.035, Market = "hk" },
        ["DJI"] = new() { Name = "道琼斯工业平均", BaseValue = 49000.0, Volatility = 0.015, Market = "us" },
        ["IXIC"] = new() { Name = "纳斯达克综合", BaseValue = 26000.0, Volatility = 0.025, Market = "us" },
        ["SPX"] = new() { Name = "标普500", BaseValue = 5800.0, Volatility = 0.018, Market = "us" },
    };

    public EastMoneyClient(HttpClient httpClient, Microsoft.Extensions.Options.IOptions<AppSettings> settings, IMemoryCache cache, ILogger<EastMoneyClient> logger)
    {
        _httpClient = httpClient;
        _settings = settings.Value.EastMoney;
        _cache = cache;
        _logger = logger;
    }

    private static string GetSecid(string code)
    {
        foreach (var prefix in ShanghaiPrefixes)
        {
            if (code.StartsWith(prefix, StringComparison.Ordinal)) return $"1.{code}";
        }
        return $"0.{code}";
    }

    public async Task<MarketIndexDto?> FetchRealtimeIndex(string code, string? tencentPrefix = null, string market = "cn", string? tencentCode = null)
    {
        var cacheKey = $"index_realtime_{code}";
        if (_cache.TryGetValue(cacheKey, out MarketIndexDto? cached))
            return cached;

        // Try Tencent Finance API first (more reliable)
        var result = await FetchRealtimeIndexFromTencent(code, tencentPrefix, market, tencentCode);
        if (result != null)
        {
            _cache.Set(cacheKey, result, TimeSpan.FromSeconds(30));
            return result;
        }

        // Fallback to EastMoney API (only for A-share indices)
        if (market == "cn")
        {
            result = await FetchRealtimeIndexFromEastMoney(code);
            if (result != null)
            {
                result.Market = market;
                _cache.Set(cacheKey, result, TimeSpan.FromSeconds(30));
                return result;
            }
        }

        return null;
    }

    private async Task<MarketIndexDto?> FetchRealtimeIndexFromTencent(string code, string? tencentPrefix = null, string market = "cn", string? tencentCode = null)
    {
        // Determine Tencent API prefix: hk for HK, us for US, sh/sz for A-share
        string prefix;
        if (!string.IsNullOrEmpty(tencentPrefix))
        {
            prefix = tencentPrefix;
        }
        else
        {
            prefix = code.StartsWith("399", StringComparison.Ordinal) ? "sz" : "sh";
        }
        // Use tencentCode if provided (e.g. ".INX" for SPX), otherwise use code
        var apiCode = !string.IsNullOrEmpty(tencentCode) ? tencentCode : code;
        var url = $"https://qt.gtimg.cn/q={prefix}{apiCode}";

        try
        {
            var response = await _httpClient.GetAsync(url);
            response.EnsureSuccessStatusCode();
            // Tencent API returns GBK charset; read as bytes then decode manually
            var bytes = await response.Content.ReadAsByteArrayAsync();
            var text = Encoding.GetEncoding("GBK").GetString(bytes);

            // Parse Tencent format: v_sh000001="1~上证指数~000001~4162.18~..."
            var eqIndex = text.IndexOf('=');
            if (eqIndex < 0) return null;

            var valuePart = text[(eqIndex + 1)..].Trim().TrimStart('"').TrimEnd('"', ';', '\n', '\r');
            if (string.IsNullOrEmpty(valuePart)) return null;

            var fields = valuePart.Split('~');
            if (fields.Length < 35) return null;

            // Field indices in Tencent format:
            // [1] name, [2] code, [3] current price, [4] prev close,
            // [31] change amount, [32] change pct, [33] high, [34] low
            var name = fields[1];
            var currentValue = double.TryParse(fields[3], NumberStyles.Any, CultureInfo.InvariantCulture, out var val) ? val : 0;
            var prevClose = double.TryParse(fields[4], NumberStyles.Any, CultureInfo.InvariantCulture, out var prev) ? prev : 0;
            var change = double.TryParse(fields[31], NumberStyles.Any, CultureInfo.InvariantCulture, out var chg) ? chg : 0;
            var changePct = double.TryParse(fields[32], NumberStyles.Any, CultureInfo.InvariantCulture, out var pct) ? pct : 0;
            var high = double.TryParse(fields[33], NumberStyles.Any, CultureInfo.InvariantCulture, out var hi) ? hi : 0;
            var low = double.TryParse(fields[34], NumberStyles.Any, CultureInfo.InvariantCulture, out var lo) ? lo : 0;

            // Data validation: current value must be positive and reasonable
            if (currentValue <= 0 || string.IsNullOrEmpty(name)) return null;

            // Cross-validate: change should be consistent with current vs prev close
            if (prevClose > 0 && Math.Abs(change - (currentValue - prevClose)) > prevClose * 0.01)
            {
                // Recalculate change from current and prev close if inconsistent
                change = Math.Round(currentValue - prevClose, 2);
                changePct = Math.Round(change / prevClose * 100, 2);
            }

            return new MarketIndexDto
            {
                Code = code,
                Name = name,
                Market = market,
                Value = currentValue,
                Change = change,
                ChangePct = changePct,
                High = high,
                Low = low,
                PrevClose = prevClose,
                // Volume field index differs by market: A-share uses fields[35], HK uses fields[37] for amount
                Volume = market == "cn" && fields.Length > 35 && double.TryParse(fields[35], NumberStyles.Any, CultureInfo.InvariantCulture, out var cnVol) ? cnVol
                       : market == "hk" && fields.Length > 37 && double.TryParse(fields[37], NumberStyles.Any, CultureInfo.InvariantCulture, out var hkAmt) ? hkAmt
                       : 0,
            };
        }
        catch (Exception ex)
        {
            _logger.LogRealtimeIndexFailed(ex, code);
        }
        return null;
    }

    private async Task<MarketIndexDto?> FetchRealtimeIndexFromEastMoney(string code)
    {
        var url = "https://push2.eastmoney.com/api/qt/stock/get";
        var secid = GetSecid(code);
        var parameters = new Dictionary<string, string>
        {
            ["secid"] = secid,
            ["fields"] = "f43,f44,f45,f46,f47,f48,f50,f51,f52,f57,f58,f169,f170",
            ["ut"] = "fa5fd1943c7b386f172d6893dbfba10b",
        };

        try
        {
            var response = await _httpClient.GetAsync(BuildUrl(url, parameters));
            response.EnsureSuccessStatusCode();
            var json = await response.Content.ReadAsStringAsync();
            using var doc = JsonDocument.Parse(json);
            var root = doc.RootElement;

            if (root.TryGetProperty("data", out var data) && data.ValueKind != JsonValueKind.Null)
            {
                var result = new MarketIndexDto
                {
                    Code = code,
                    Name = TryGetString(data, "f58"),
                    Value = TryGetDouble(data, "f43") / 100.0,
                    Change = TryGetDouble(data, "f169") / 100.0,
                    ChangePct = TryGetDouble(data, "f170") / 100.0,
                };

                // Data validation
                if (result.Value <= 0 || string.IsNullOrEmpty(result.Name)) return null;

                return result;
            }
        }
        catch (Exception ex)
        {
            _logger.LogRealtimeIndexFailed(ex, code);
        }
        return null;
    }

    public async Task<List<double>> FetchIndexMinichart(string code, string market = "cn")
    {
        var cacheKey = $"index_minichart_{code}";
        if (_cache.TryGetValue(cacheKey, out List<double>? cached))
            return cached!;

        // HK/US indices: skip minichart API (not supported), generate simulated chart
        if (market == "hk" || market == "us")
        {
            var simData = SimulatedIndexData.GetValueOrDefault(code);
            if (simData != null)
            {
                var chartData = GenerateIntradayChart(simData.BaseValue, 0);
                _cache.Set(cacheKey, chartData, TimeSpan.FromMinutes(5));
                return chartData;
            }
            return new List<double>();
        }

        // A-share: Try Tencent Finance API first
        var result = await FetchIndexMinichartFromTencent(code);
        if (result.Count > 0)
        {
            _cache.Set(cacheKey, result, TimeSpan.FromSeconds(30));
            return result;
        }

        // Fallback to EastMoney API
        result = await FetchIndexMinichartFromEastMoney(code);
        if (result.Count > 0)
        {
            _cache.Set(cacheKey, result, TimeSpan.FromSeconds(30));
            return result;
        }

        return result;
    }

    private async Task<List<double>> FetchIndexMinichartFromTencent(string code)
    {
        var prefix = code.StartsWith("399", StringComparison.Ordinal) ? "sz" : "sh";
        var url = $"https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param={prefix}{code},day,,,20,qfq";

        try
        {
            var response = await _httpClient.GetAsync(url);
            response.EnsureSuccessStatusCode();
            var bytes = await response.Content.ReadAsByteArrayAsync();
            var json = Encoding.GetEncoding("GBK").GetString(bytes);
            using var doc = JsonDocument.Parse(json);
            var root = doc.RootElement;

            // Navigate: data -> {prefix}{code} -> qfqday or day
            if (!root.TryGetProperty("data", out var data)) return new List<double>();
            var stockKey = $"{prefix}{code}";
            if (!data.TryGetProperty(stockKey, out var stockData)) return new List<double>();

            // Try qfqday first (前复权), then day
            var dayKey = stockData.TryGetProperty("qfqday", out var qfqday) ? "qfqday" : "day";
            if (!stockData.TryGetProperty(dayKey, out var dayArr)) return new List<double>();

            var prices = new List<double>();
            foreach (var item in dayArr.EnumerateArray())
            {
                if (item.ValueKind == JsonValueKind.Array)
                {
                    var arr = item.EnumerateArray().ToArray();
                    // Format: [date, open, close, high, low, ...]
                    if (arr.Length >= 3 && double.TryParse(arr[2].GetString(), NumberStyles.Any, CultureInfo.InvariantCulture, out var closePrice))
                    {
                        prices.Add(closePrice);
                    }
                }
            }

            if (prices.Count >= 20)
            {
                var step = prices.Count / 20.0;
                var sampled = new List<double>();
                for (int i = 0; i < 20; i++)
                {
                    sampled.Add(Math.Round(prices[(int)(i * step)], 2));
                }
                return sampled;
            }

            return prices.Select(p => Math.Round(p, 2)).ToList();
        }
        catch (Exception ex)
        {
            _logger.LogMinichartFailed(ex, code);
        }
        return new List<double>();
    }

    private async Task<List<double>> FetchIndexMinichartFromEastMoney(string code)
    {
        var secid = GetSecid(code);
        var url = "https://push2his.eastmoney.com/api/qt/stock/trends2/get";
        var parameters = new Dictionary<string, string>
        {
            ["secid"] = secid,
            ["fields1"] = "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
            ["fields2"] = "f51,f52,f53,f54,f55,f56,f57,f58",
            ["iscr"] = "0",
            ["ndays"] = "1",
            ["ut"] = "fa5fd1943c7b386f172d6893dbfba10b",
        };

        try
        {
            var response = await _httpClient.GetAsync(BuildUrl(url, parameters));
            response.EnsureSuccessStatusCode();
            var json = await response.Content.ReadAsStringAsync();
            using var doc = JsonDocument.Parse(json);
            var root = doc.RootElement;

            if (!root.TryGetProperty("data", out var data) || data.ValueKind == JsonValueKind.Null)
                return new List<double>();
            if (!data.TryGetProperty("trends", out var trends) || trends.ValueKind == JsonValueKind.Null)
                return new List<double>();

            var prices = new List<double>();
            foreach (var t in trends.EnumerateArray())
            {
                var parts = t.GetString()!.Split(',');
                if (parts.Length >= 2 && double.TryParse(parts[1], out var price))
                {
                    prices.Add(price);
                }
            }

            List<double> result;
            if (prices.Count >= 20)
            {
                var step = prices.Count / 20.0;
                var sampled = new List<double>();
                for (int i = 0; i < 20; i++)
                {
                    sampled.Add(Math.Round(prices[(int)(i * step)], 2));
                }
                result = sampled;
            }
            else if (prices.Count >= 2)
            {
                result = prices.Select(p => Math.Round(p, 2)).ToList();
            }
            else
            {
                result = new List<double>();
            }

            return result;
        }
        catch (Exception ex)
        {
            _logger.LogMinichartFailed(ex, code);
        }
        return new List<double>();
    }

    public async Task<Dictionary<string, object>?> FetchFundRealtime(string fundCode)
    {
        var cacheKey = $"fund_realtime_{fundCode}";
        if (_cache.TryGetValue(cacheKey, out Dictionary<string, object>? cached))
            return cached;

        // Primary: Tencent Finance API (more reliable, provides real NAV + daily change)
        var result = await FetchFundRealtimeFromTencent(fundCode);
        if (result != null)
        {
            _cache.Set(cacheKey, result, GetCacheTtl());
            return result;
        }

        // Fallback: 天天基金估值接口
        result = await FetchFundRealtimeFrom1234567(fundCode);
        if (result != null)
        {
            _cache.Set(cacheKey, result, GetCacheTtl());
            return result;
        }

        return null;
    }

    /// <summary>
    /// 批量获取基金实时数据 — 一次HTTP请求获取多个基金，大幅减少网络延迟
    /// 腾讯API支持: q=jj000001,jj110011,jj161725
    /// </summary>
    public async Task<Dictionary<string, Dictionary<string, object>>> FetchFundRealtimeBatch(IEnumerable<string> fundCodes)
    {
        var result = new Dictionary<string, Dictionary<string, object>>();
        var codesList = fundCodes.ToList();

        // Check cache first, collect uncached codes
        var uncachedCodes = new List<string>();
        foreach (var code in codesList)
        {
            var cacheKey = $"fund_realtime_{code}";
            if (_cache.TryGetValue(cacheKey, out Dictionary<string, object>? cached) && cached != null)
            {
                result[code] = cached;
            }
            else
            {
                uncachedCodes.Add(code);
            }
        }

        if (uncachedCodes.Count == 0) return result;

        // Batch fetch from Tencent API (max 50 per request)
        foreach (var batch in uncachedCodes.Chunk(50))
        {
            var batchResult = await FetchFundRealtimeBatchFromTencent(batch);
            foreach (var kvp in batchResult)
            {
                result[kvp.Key] = kvp.Value;
                _cache.Set($"fund_realtime_{kvp.Key}", kvp.Value, GetCacheTtl());
            }

            // Fallback: fetch individually for codes not returned by batch
            var missingCodes = batch.Where(c => !batchResult.ContainsKey(c)).ToList();
            foreach (var code in missingCodes)
            {
                var single = await FetchFundRealtimeFrom1234567(code);
                if (single != null)
                {
                    result[code] = single;
                    _cache.Set($"fund_realtime_{code}", single, GetCacheTtl());
                }
            }
        }

        return result;
    }

    /// <summary>
    /// 腾讯批量基金查询 — 一次请求获取多个基金数据
    /// </summary>
    private async Task<Dictionary<string, Dictionary<string, object>>> FetchFundRealtimeBatchFromTencent(string[] fundCodes)
    {
        var result = new Dictionary<string, Dictionary<string, object>>();
        var query = string.Join(",", fundCodes.Select(c => $"jj{c}"));
        var url = $"https://qt.gtimg.cn/q={query}";

        try
        {
            var response = await _httpClient.GetAsync(url);
            response.EnsureSuccessStatusCode();
            var bytes = await response.Content.ReadAsByteArrayAsync();
            var text = Encoding.GetEncoding("GBK").GetString(bytes);

            // Parse multiple lines: v_jj000001="...";v_jj110011="...";
            var lines = text.Split(';', StringSplitOptions.RemoveEmptyEntries);
            foreach (var line in lines)
            {
                var eqIndex = line.IndexOf('=');
                if (eqIndex < 0) continue;

                var valuePart = line[(eqIndex + 1)..].Trim().TrimStart('"').TrimEnd('"', '\n', '\r');
                if (string.IsNullOrEmpty(valuePart)) continue;

                var fields = valuePart.Split('~');
                if (fields.Length < 9) continue;

                var code = fields[0];
                var latestNav = double.TryParse(fields[5], NumberStyles.Any, CultureInfo.InvariantCulture, out var nav) ? nav : 0;
                var cumulativeNav = double.TryParse(fields[6], NumberStyles.Any, CultureInfo.InvariantCulture, out var cnav) ? cnav : 0;
                var dailyChangePct = double.TryParse(fields[7], NumberStyles.Any, CultureInfo.InvariantCulture, out var dcpct) ? dcpct : 0;
                var navDate = fields.Length > 8 ? fields[8] : "";

                var estimatedNav = double.TryParse(fields[2], NumberStyles.Any, CultureInfo.InvariantCulture, out var enav) ? enav : 0;
                var estimatedChangePct = double.TryParse(fields[3], NumberStyles.Any, CultureInfo.InvariantCulture, out var epct) ? epct : 0;

                var displayNav = estimatedNav > 0 ? estimatedNav : latestNav;
                var displayChangePct = estimatedChangePct != 0 ? estimatedChangePct : dailyChangePct;

                if (displayNav <= 0 && latestNav <= 0) continue;

                result[code] = new Dictionary<string, object>
                {
                    ["fund_code"] = code,
                    ["fund_name"] = fields[1],
                    ["estimated_nav"] = displayNav > 0 ? displayNav : latestNav,
                    ["latest_nav"] = latestNav,
                    ["cumulative_nav"] = cumulativeNav,
                    ["change_pct"] = displayChangePct,
                    ["daily_change_pct"] = dailyChangePct,
                    ["nav_date"] = navDate,
                };
            }
        }
        catch (Exception ex)
        {
            _logger.LogFundRealtimeFailed(ex, string.Join(",", fundCodes));
        }

        return result;
    }

    /// <summary>
    /// 动态缓存TTL: 交易时段(9:30-15:00)30秒，非交易时段300秒
    /// </summary>
    private static TimeSpan GetCacheTtl()
    {
        var now = DateTime.Now;
        var time = now.TimeOfDay;
        var isTradingHours = time >= new TimeSpan(9, 30, 0) && time <= new TimeSpan(15, 0, 0)
            && now.DayOfWeek != DayOfWeek.Saturday && now.DayOfWeek != DayOfWeek.Sunday;
        return isTradingHours ? TimeSpan.FromSeconds(30) : TimeSpan.FromSeconds(300);
    }

    /// <summary>
    /// 腾讯财经基金接口 — 提供真实单位净值、累计净值和日涨跌幅
    /// 格式: v_jj{code}="code~名称~估算净值~估算涨跌幅~~单位净值~累计净值~日涨跌幅~日期"
    /// </summary>
    private async Task<Dictionary<string, object>?> FetchFundRealtimeFromTencent(string fundCode)
    {
        var url = $"https://qt.gtimg.cn/q=jj{fundCode}";
        try
        {
            var response = await _httpClient.GetAsync(url);
            response.EnsureSuccessStatusCode();
            var bytes = await response.Content.ReadAsByteArrayAsync();
            var text = Encoding.GetEncoding("GBK").GetString(bytes);

            // Parse: v_jj000001="000001~华夏成长混合~0.0000~0.0000~~1.3190~3.8920~-0.1239~2026-05-20~";
            var eqIndex = text.IndexOf('=');
            if (eqIndex < 0) return null;

            var valuePart = text[(eqIndex + 1)..].Trim().TrimStart('"').TrimEnd('"', ';', '\n', '\r');
            if (string.IsNullOrEmpty(valuePart)) return null;

            var fields = valuePart.Split('~');
            if (fields.Length < 9) return null;

            // fields[5] = 单位净值, fields[6] = 累计净值, fields[7] = 日涨跌幅, fields[8] = 净值日期
            var latestNav = double.TryParse(fields[5], NumberStyles.Any, CultureInfo.InvariantCulture, out var nav) ? nav : 0;
            var cumulativeNav = double.TryParse(fields[6], NumberStyles.Any, CultureInfo.InvariantCulture, out var cnav) ? cnav : 0;
            var dailyChangePct = double.TryParse(fields[7], NumberStyles.Any, CultureInfo.InvariantCulture, out var dcpct) ? dcpct : 0;
            var navDate = fields.Length > 8 ? fields[8] : "";

            // Also try estimated values (fields[2], fields[3]) for intraday
            var estimatedNav = double.TryParse(fields[2], NumberStyles.Any, CultureInfo.InvariantCulture, out var enav) ? enav : 0;
            var estimatedChangePct = double.TryParse(fields[3], NumberStyles.Any, CultureInfo.InvariantCulture, out var epct) ? epct : 0;

            // Use estimated values if available (trading hours), otherwise use real NAV
            var displayNav = estimatedNav > 0 ? estimatedNav : latestNav;
            var displayChangePct = estimatedChangePct != 0 ? estimatedChangePct : dailyChangePct;

            if (displayNav <= 0 && latestNav <= 0) return null;

            var result = new Dictionary<string, object>
            {
                ["fund_code"] = fundCode,
                ["fund_name"] = fields[1],
                ["estimated_nav"] = displayNav > 0 ? displayNav : latestNav,
                ["latest_nav"] = latestNav,
                ["cumulative_nav"] = cumulativeNav,
                ["change_pct"] = displayChangePct,
                ["daily_change_pct"] = dailyChangePct,
                ["nav_date"] = navDate,
            };
            return result;
        }
        catch (Exception ex)
        {
            _logger.LogFundRealtimeFailed(ex, fundCode);
        }
        return null;
    }

    /// <summary>
    /// 天天基金估值接口 — 仅交易时间有估值数据
    /// </summary>
    private async Task<Dictionary<string, object>?> FetchFundRealtimeFrom1234567(string fundCode)
    {
        var url = $"https://fundgz.1234567.com.cn/js/{fundCode}.js";
        try
        {
            var response = await _httpClient.GetAsync(url);
            response.EnsureSuccessStatusCode();
            var text = await response.Content.ReadAsStringAsync();

            if (text.Contains("jsonpgz("))
            {
                var start = text.IndexOf('(') + 1;
                var end = text.LastIndexOf(')');
                var jsonStr = text[start..end];
                using var doc = JsonDocument.Parse(jsonStr);
                var root = doc.RootElement;

                var result = new Dictionary<string, object>
                {
                    ["fund_code"] = TryGetString(root, "fundcode"),
                    ["fund_name"] = TryGetString(root, "name"),
                    ["estimated_nav"] = double.TryParse(TryGetString(root, "gsz"), out var gsz) ? gsz : 0,
                    ["change_pct"] = double.TryParse(TryGetString(root, "gszzl"), out var gszzl) ? gszzl : 0,
                    ["nav_date"] = TryGetString(root, "gztime"),
                };
                return result;
            }
        }
        catch (Exception ex)
        {
            _logger.LogFundRealtimeFailed(ex, fundCode);
        }
        return null;
    }

    public static List<double> GenerateIntradayChart(double currentValue, double changePct, int numPoints = 20)
    {
        var prevClose = changePct != 0 ? currentValue / (1 + changePct / 100) : currentValue;
        var chart = new List<double>();
        var dailyRange = Math.Abs(currentValue - prevClose);

        for (int i = 0; i < numPoints; i++)
        {
            var progress = (double)i / (numPoints - 1);
            var basePoint = prevClose + (currentValue - prevClose) * progress;
            var noise = Random.Shared.NextDouble() * dailyRange * 0.3 - dailyRange * 0.15;
            chart.Add(Math.Round(basePoint + noise, 2));
        }
        return chart;
    }

    public MarketIndexDto GenerateSimulatedIndex(string code)
    {
        var simData = SimulatedIndexData.GetValueOrDefault(code, new SimulatedIndexData { Name = "未知指数", BaseValue = 3000.0, Volatility = 0.02, Market = "cn" });
        var changePct = Math.Round(Random.Shared.NextDouble() * simData.Volatility * 200 - simData.Volatility * 100, 2);
        var value = Math.Round(simData.BaseValue * (1 + changePct / 100), 2);
        var change = Math.Round(simData.BaseValue * changePct / 100, 2);
        var chartData = GenerateIntradayChart(value, changePct);

        return new MarketIndexDto
        {
            Code = code,
            Name = simData.Name,
            Market = simData.Market,
            Value = value,
            Change = change,
            ChangePct = changePct,
            MiniChartData = chartData,
            UpdateTime = DateTime.Now.ToString("o"),
            DataSource = "simulated",
        };
    }

    public FundSearchDataDto SearchFunds(string keyword, int page = 1, int size = 20)
    {
        var matched = BuiltinFunds
            .Where(f => f.FundCode.Contains(keyword, StringComparison.OrdinalIgnoreCase) || f.FundName.Contains(keyword, StringComparison.OrdinalIgnoreCase))
            .ToList();

        var total = matched.Count;
        var start = (page - 1) * size;
        var items = matched.Skip(start).Take(size).ToList();

        return new FundSearchDataDto
        {
            Items = items,
            Total = total,
            Page = page,
            Size = size,
        };
    }

    public FundItemDto? GetFundByCode(string fundCode)
    {
        return BuiltinFunds.FirstOrDefault(f => f.FundCode == fundCode);
    }

    public List<string> GetAllFundCodes()
    {
        return BuiltinFunds.Select(f => f.FundCode).ToList();
    }

    private static string BuildUrl(string baseUrl, Dictionary<string, string> parameters)
    {
        var query = string.Join("&", parameters.Select(p => $"{p.Key}={Uri.EscapeDataString(p.Value)}"));
        return $"{baseUrl}?{query}";
    }

    private static string TryGetString(JsonElement element, string propertyName)
    {
        if (element.TryGetProperty(propertyName, out var prop) && prop.ValueKind == JsonValueKind.String)
            return prop.GetString() ?? "";
        if (element.TryGetProperty(propertyName, out prop) && prop.ValueKind == JsonValueKind.Number)
            return prop.GetDouble().ToString(CultureInfo.InvariantCulture);
        return "";
    }

    private static double TryGetDouble(JsonElement element, string propertyName)
    {
        if (element.TryGetProperty(propertyName, out var prop))
        {
            if (prop.ValueKind == JsonValueKind.Number) return prop.GetDouble();
            if (prop.ValueKind == JsonValueKind.String && double.TryParse(prop.GetString(), out var val)) return val;
        }
        return 0;
    }
}

public sealed class MarketIndexConfig
{
    public string Code { get; set; } = "";
    public string Name { get; set; } = "";
    public string Market { get; set; } = "cn";
    public string? TencentPrefix { get; set; }
    public string? TencentCode { get; set; } // Override code for Tencent API (e.g. ".INX" for SPX)
}

public sealed class SimulatedIndexData
{
    public string Name { get; set; } = "";
    public double BaseValue { get; set; }
    public double Volatility { get; set; }
    public string Market { get; set; } = "cn";
}

internal static partial class EastMoneyLogMessages
{
    [LoggerMessage(Level = LogLevel.Warning, Message = "fetch_realtime_index failed for {Code}")]
    public static partial void LogRealtimeIndexFailed(this ILogger logger, Exception ex, string code);

    [LoggerMessage(Level = LogLevel.Warning, Message = "fetch_index_minichart failed for {Code}")]
    public static partial void LogMinichartFailed(this ILogger logger, Exception ex, string code);

    [LoggerMessage(Level = LogLevel.Debug, Message = "fetch_fund_realtime failed for {FundCode}")]
    public static partial void LogFundRealtimeFailed(this ILogger logger, Exception ex, string fundCode);
}
