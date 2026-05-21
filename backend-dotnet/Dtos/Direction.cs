using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public enum Direction
{
    [JsonPropertyName("up")]
    Up,
    [JsonPropertyName("down")]
    Down,
    [JsonPropertyName("flat")]
    Flat
}
