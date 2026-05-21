using Microsoft.Extensions.Logging;

namespace StockPredictApi.Middleware;

public class GlobalExceptionMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ILogger<GlobalExceptionMiddleware> _logger;
    private readonly bool _isProduction;

    public GlobalExceptionMiddleware(RequestDelegate next, ILogger<GlobalExceptionMiddleware> logger, IConfiguration configuration)
    {
        _next = next;
        _logger = logger;
        _isProduction = configuration["AppSettings:Env"] == "production";
    }

    public async Task InvokeAsync(HttpContext context)
    {
        try
        {
            await _next(context);
        }
        catch (OperationCanceledException) when (context.RequestAborted.IsCancellationRequested)
        {
            // Client disconnected — not an error, skip logging
        }
        catch (ArgumentException ex)
        {
            if (context.Response.HasStarted)
            {
                _logger.LogValidationError(ex, ex.Message);
                return;
            }
            _logger.LogValidationError(ex, ex.Message);
            context.Response.StatusCode = 400;
            await context.Response.WriteAsJsonAsync(new { code = -1, message = ex.Message, data = (object?)null });
        }
        catch (Exception ex)
        {
            if (context.Response.HasStarted)
            {
                _logger.LogUnhandledException(ex);
                return;
            }
            _logger.LogUnhandledException(ex);
            var msg = _isProduction ? "服务器内部错误" : ex.Message;
            context.Response.StatusCode = 500;
            await context.Response.WriteAsJsonAsync(new { code = -1, message = msg, data = (object?)null });
        }
    }
}

internal static partial class GlobalExceptionLogMessages
{
    [LoggerMessage(Level = LogLevel.Warning, Message = "Validation error: {Message}")]
    public static partial void LogValidationError(this ILogger logger, Exception ex, string message);

    [LoggerMessage(Level = LogLevel.Error, Message = "Unhandled exception")]
    public static partial void LogUnhandledException(this ILogger logger, Exception ex);
}
