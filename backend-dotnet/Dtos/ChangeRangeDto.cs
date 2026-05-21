using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class ChangeRangeDto
{
    [JsonPropertyName("low")] public double Low { get; set; }
    [JsonPropertyName("high")] public double High { get; set; }
}
