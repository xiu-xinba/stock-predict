using Microsoft.AspNetCore.Mvc;
using StockPredictApi.Services;

namespace StockPredictApi.Controllers;

[ApiController]
[Route("api/v1")]
public class HealthController : ControllerBase
{
    private readonly PredictionService _predictionService;

    public HealthController(PredictionService predictionService)
    {
        _predictionService = predictionService;
    }

    [HttpGet("health")]
    public IActionResult HealthCheck()
    {
        return Ok(new { status = "ok", model_loaded = _predictionService.IsModelLoaded() });
    }
}
