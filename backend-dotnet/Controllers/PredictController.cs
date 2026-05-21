using Microsoft.AspNetCore.Mvc;
using StockPredictApi.Dtos;
using StockPredictApi.Services;

namespace StockPredictApi.Controllers;

[ApiController]
[Route("api/v1")]
public class PredictController : ControllerBase
{
    private readonly PredictionService _predictionService;

    public PredictController(PredictionService predictionService)
    {
        _predictionService = predictionService;
    }

    [HttpGet("predict/{fundCode}")]
    public async Task<ActionResult<ApiResponse<PredictionDataDto>>> Predict(string fundCode)
    {
        var result = await _predictionService.Predict(fundCode);
        return Ok(result);
    }
}
