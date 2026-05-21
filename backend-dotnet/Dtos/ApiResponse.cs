using System.Text.Json.Serialization;

namespace StockPredictApi.Dtos;

public class ApiResponse
{
    [JsonPropertyName("code")]
    public int Code { get; set; } = 0;

    [JsonPropertyName("message")]
    public string Message { get; set; } = "success";

    public static ApiResponse<T> Success<T>(T data, string message = "success") => new() { Code = 0, Message = message, Data = data };
    public static ApiResponse<T> Error<T>(int code, string message) => new() { Code = code, Message = message, Data = default };
}

public class ApiResponse<T> : ApiResponse
{
    [JsonPropertyName("data")]
    public T? Data { get; set; }
}
