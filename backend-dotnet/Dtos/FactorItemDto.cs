using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class FactorItemDto
{
    [JsonPropertyName("name")] public string Name { get; set; } = "";
    [JsonPropertyName("importance")] public double Importance { get; set; }
    [JsonPropertyName("description")] public string Description { get; set; } = "";
}
