using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class MarketSnapshotDto
{
    [JsonPropertyName("sh_index")] public double ShIndex { get; set; }
    [JsonPropertyName("sh_index_change_pct")] public double ShIndexChangePct { get; set; }
    [JsonPropertyName("sz_index")] public double SzIndex { get; set; }
    [JsonPropertyName("sz_index_change_pct")] public double SzIndexChangePct { get; set; }
    [JsonPropertyName("cyb_index")] public double CybIndex { get; set; }
    [JsonPropertyName("cyb_index_change_pct")] public double CybIndexChangePct { get; set; }
    [JsonPropertyName("update_time")] public string UpdateTime { get; set; } = "";
}
