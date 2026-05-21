using System.Globalization;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Caching.Memory;
using StockPredictApi.Data;
using StockPredictApi.Dtos;
using StockPredictApi.Models;

namespace StockPredictApi.Services;

public class FundService
{
    private readonly AppDbContext _db;
    private readonly IMemoryCache _cache;

    public FundService(AppDbContext db, IMemoryCache cache)
    {
        _db = db;
        _cache = cache;
    }

    public async Task<ApiResponse<FundSearchDataDto>> SearchFundsAsync(
        string keyword,
        string? type = null,
        string? company = null,
        string? riskLevel = null,
        string? manager = null,
        double? returnMin = null,
        double? returnMax = null,
        string sortBy = "relevance",
        string sortOrder = "desc",
        int page = 1,
        int size = 20)
    {
        var cacheKey = $"fund_search_{keyword}_{type}_{company}_{riskLevel}_{manager}_{returnMin?.ToString(CultureInfo.InvariantCulture)}_{returnMax?.ToString(CultureInfo.InvariantCulture)}_{sortBy}_{sortOrder}_{page}_{size}";
        if (_cache.TryGetValue(cacheKey, out ApiResponse<FundSearchDataDto>? cached))
            return cached!;

        var query = _db.Funds.AsQueryable();

        // Keyword filter (code or name)
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            query = query.Where(f => f.FundCode.Contains(keyword) || f.FundName.Contains(keyword));
        }

        // Type filter
        if (!string.IsNullOrWhiteSpace(type))
        {
            query = query.Where(f => f.FundType == type);
        }

        // Company filter
        if (!string.IsNullOrWhiteSpace(company))
        {
            query = query.Where(f => f.Company == company);
        }

        // Risk level filter
        if (!string.IsNullOrWhiteSpace(riskLevel))
        {
            query = query.Where(f => f.RiskLevel == riskLevel);
        }

        // Manager filter
        if (!string.IsNullOrWhiteSpace(manager))
        {
            query = query.Where(f => f.Manager == manager);
        }

        // Return range filter (1Y)
        if (returnMin.HasValue)
        {
            query = query.Where(f => f.Return1Y >= returnMin.Value);
        }
        if (returnMax.HasValue)
        {
            query = query.Where(f => f.Return1Y <= returnMax.Value);
        }

        // Sorting
        query = ApplySorting(query, keyword, sortBy, sortOrder);

        var total = await query.CountAsync();
        var items = await query
            .Skip((page - 1) * size)
            .Take(size)
            .Select(f => MapToDto(f))
            .ToListAsync();

        var result = ApiResponse.Success(new FundSearchDataDto
        {
            Items = items,
            Total = total,
            Page = page,
            Size = size,
        });

        // Cache for 60 seconds
        _cache.Set(cacheKey, result, TimeSpan.FromSeconds(60));

        return result;
    }

    public async Task<ApiResponse<FundItemDto>> GetFundAsync(string fundCode)
    {
        var fund = await _db.Funds.FindAsync(fundCode);
        if (fund == null)
            return ApiResponse.Error<FundItemDto>(-1, $"未找到基金 {fundCode}");
        return ApiResponse.Success(MapToDto(fund));
    }

    public async Task<ApiResponse<FundFiltersDto>> GetFiltersAsync()
    {
        const string cacheKey = "fund_filters";
        if (_cache.TryGetValue(cacheKey, out ApiResponse<FundFiltersDto>? cached))
            return cached!;

        var types = await _db.Funds
            .Where(f => f.FundType != null)
            .Select(f => f.FundType)
            .Distinct()
            .OrderBy(t => t)
            .ToListAsync();

        var companies = await _db.Funds
            .Where(f => f.Company != null)
            .Select(f => f.Company!)
            .Distinct()
            .OrderBy(c => c)
            .ToListAsync();

        var riskLevels = await _db.Funds
            .Where(f => f.RiskLevel != null)
            .Select(f => f.RiskLevel!)
            .Distinct()
            .OrderBy(r => r)
            .ToListAsync();

        var result = ApiResponse.Success(new FundFiltersDto
        {
            Types = types,
            Companies = companies,
            RiskLevels = riskLevels,
        });

        _cache.Set(cacheKey, result, TimeSpan.FromSeconds(300));

        return result;
    }

    private static IQueryable<Fund> ApplySorting(IQueryable<Fund> query, string keyword, string sortBy, string sortOrder)
    {
        var isDesc = sortOrder.Equals("desc", StringComparison.OrdinalIgnoreCase);

        // For relevance sorting with keyword, prioritize exact code matches
        if (sortBy == "relevance" && !string.IsNullOrWhiteSpace(keyword))
        {
            return query.OrderByDescending(f => f.FundCode == keyword ? 2 : (f.FundCode.StartsWith(keyword) ? 1 : 0))
                        .ThenBy(f => f.FundCode);
        }

        return sortBy switch
        {
            "return_1y" => isDesc ? query.OrderByDescending(f => f.Return1Y) : query.OrderBy(f => f.Return1Y),
            "return_3y" => isDesc ? query.OrderByDescending(f => f.Return3Y) : query.OrderBy(f => f.Return3Y),
            "latest_nav" => isDesc ? query.OrderByDescending(f => f.LatestNav) : query.OrderBy(f => f.LatestNav),
            "inception_date" => isDesc ? query.OrderByDescending(f => f.InceptionDate) : query.OrderBy(f => f.InceptionDate),
            _ => query.OrderBy(f => f.FundCode),
        };
    }

    private static FundItemDto MapToDto(Fund f) => new()
    {
        FundCode = f.FundCode,
        FundName = f.FundName,
        FundType = f.FundType,
        Company = f.Company,
        Manager = f.Manager,
        LatestNav = f.LatestNav,
        CumulativeNav = f.CumulativeNav,
        Return1M = f.Return1M,
        Return3M = f.Return3M,
        Return6M = f.Return6M,
        Return1Y = f.Return1Y,
        Return3Y = f.Return3Y,
        RiskLevel = f.RiskLevel,
        InceptionDate = f.InceptionDate,
    };
}
