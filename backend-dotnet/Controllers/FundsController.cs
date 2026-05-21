using Microsoft.AspNetCore.Mvc;
using StockPredictApi.Dtos;
using StockPredictApi.Services;

namespace StockPredictApi.Controllers;

[ApiController]
[Route("api/v1/funds")]
public class FundsController : ControllerBase
{
    private readonly FundService _fundService;
    private readonly FundSyncService _fundSyncService;

    public FundsController(FundService fundService, FundSyncService fundSyncService)
    {
        _fundService = fundService;
        _fundSyncService = fundSyncService;
    }

    /// <summary>
    /// 搜索基金（支持筛选/排序/分页）
    /// </summary>
    [HttpGet("search")]
    public async Task<ActionResult<ApiResponse<FundSearchDataDto>>> Search(
        [FromQuery] string keyword = "",
        [FromQuery] string? type = null,
        [FromQuery] string? company = null,
        [FromQuery] string? risk_level = null,
        [FromQuery] string? manager = null,
        [FromQuery] double? return_min = null,
        [FromQuery] double? return_max = null,
        [FromQuery] string sort_by = "relevance",
        [FromQuery] string sort_order = "desc",
        [FromQuery] int page = 1,
        [FromQuery] int size = 20)
    {
        if (page < 1) page = 1;
        if (size < 1) size = 1;
        if (size > 50) size = 50;

        var result = await _fundService.SearchFundsAsync(
            keyword, type, company, risk_level, manager,
            return_min, return_max, sort_by, sort_order, page, size);
        return Ok(result);
    }

    /// <summary>
    /// 获取筛选选项（基金类型/公司/风险等级）
    /// </summary>
    [HttpGet("filters")]
    public async Task<ActionResult<ApiResponse<FundFiltersDto>>> GetFilters()
    {
        var result = await _fundService.GetFiltersAsync();
        return Ok(result);
    }

    /// <summary>
    /// 手动触发基金数据同步（管理员）
    /// </summary>
    [HttpPost("sync")]
    public async Task<ActionResult<ApiResponse<int>>> TriggerSync()
    {
        var total = await _fundSyncService.FullSyncAsync(HttpContext.RequestAborted);
        return Ok(ApiResponse.Success(total));
    }
}
