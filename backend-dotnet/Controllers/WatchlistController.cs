using Microsoft.AspNetCore.Mvc;
using StockPredictApi.Dtos;
using StockPredictApi.Services;

namespace StockPredictApi.Controllers;

[ApiController]
[Route("api/v1/watchlist")]
public class WatchlistController : ControllerBase
{
    private readonly PredictionService _predictionService;

    public WatchlistController(PredictionService predictionService)
    {
        _predictionService = predictionService;
    }

    [HttpPost("quotes")]
    public async Task<ActionResult<ApiResponse<List<WatchlistItemDto>>>> GetQuotes([FromBody] WatchlistQuoteRequestDto request)
    {
        if (request.Codes == null || request.Codes.Count == 0)
            return Ok(ApiResponse.Success(new List<WatchlistItemDto>()));

        if (request.Codes.Count > 50)
            return BadRequest(ApiResponse.Error<List<WatchlistItemDto>>(-1, "最多支持50个基金代码"));

        var result = await _predictionService.GetWatchlistQuotes(request.Codes);
        return Ok(result);
    }
}
