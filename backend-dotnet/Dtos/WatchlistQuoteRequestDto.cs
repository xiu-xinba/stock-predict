using System.ComponentModel.DataAnnotations;
using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class WatchlistQuoteRequestDto
{
    [JsonPropertyName("codes")]
    [MaxLength(50, ErrorMessage = "最多支持50个基金代码")]
    public List<string> Codes { get; set; } = new();
}
