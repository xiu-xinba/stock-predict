using Microsoft.AspNetCore.Mvc;
using StockPredictApi.Dtos;
using StockPredictApi.Services;

namespace StockPredictApi.Controllers;

[ApiController]
[Route("api/v1/market")]
public class MarketController : ControllerBase
{
    private readonly MarketService _marketService;

    public MarketController(MarketService marketService)
    {
        _marketService = marketService;
    }

    [HttpGet("indices")]
    public async Task<ActionResult<ApiResponse<List<MarketIndexDto>>>> GetIndices()
    {
        var result = await _marketService.GetIndices();
        return Ok(result);
    }

    [HttpGet("ranking/{rankingType}")]
    public ActionResult<ApiResponse<List<FundRankingItemDto>>> GetRanking(string rankingType, [FromQuery] int size = 5)
    {
        if (rankingType != "gainers" && rankingType != "losers")
            return Ok(ApiResponse.Error<List<FundRankingItemDto>>(-1, "type 必须为 gainers 或 losers"));

        if (size < 1) size = 1;
        if (size > 50) size = 50;

        var result = _marketService.GetRanking(rankingType, size);
        return Ok(result);
    }
}
