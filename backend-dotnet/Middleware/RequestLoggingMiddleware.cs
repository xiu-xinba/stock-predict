using Microsoft.Extensions.Logging;

namespace StockPredictApi.Middleware;

public class RequestLoggingMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ILogger<RequestLoggingMiddleware> _logger;

    public RequestLoggingMiddleware(RequestDelegate next, ILogger<RequestLoggingMiddleware> logger)
    {
        _next = next;
        _logger = logger;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        var sw = System.Diagnostics.Stopwatch.StartNew();
        await _next(context);
        sw.Stop();
        _logger.LogRequest(context.Request.Method, context.Request.Path, context.Response.StatusCode, sw.ElapsedMilliseconds);
    }
}

internal static partial class RequestLoggingLogMessages
{
    [LoggerMessage(Level = LogLevel.Information, Message = "{Method} {Path} -> {StatusCode} ({ElapsedMs}ms)")]
    public static partial void LogRequest(this ILogger logger, string method, string path, int statusCode, long elapsedMs);
}
