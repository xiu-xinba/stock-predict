using System.IO.Compression;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Microsoft.AspNetCore.ResponseCompression;
using Microsoft.EntityFrameworkCore;
using Polly;
using StockPredictApi.Configuration;
using StockPredictApi.Data;
using StockPredictApi.Dtos;
using StockPredictApi.Services;
using StockPredictApi.Clients;
using StockPredictApi.Middleware;

// Register GBK encoding provider for Tencent Finance API
Encoding.RegisterProvider(CodePagesEncodingProvider.Instance);

var builder = WebApplication.CreateBuilder(args);

// Configuration
builder.Services.Configure<AppSettings>(builder.Configuration.GetSection("AppSettings"));
var appSettings = builder.Configuration.GetSection("AppSettings").Get<AppSettings>()!;

// Response Compression (performance optimization)
builder.Services.AddResponseCompression(options =>
{
    options.EnableForHttps = true;
    options.Providers.Add<BrotliCompressionProvider>();
    options.Providers.Add<GzipCompressionProvider>();
});
builder.Services.Configure<BrotliCompressionProviderOptions>(options => options.Level = CompressionLevel.Fastest);
builder.Services.Configure<GzipCompressionProviderOptions>(options => options.Level = CompressionLevel.Fastest);

// CORS
builder.Services.AddCors(options =>
{
    options.AddDefaultPolicy(policy =>
    {
        var origins = appSettings.CorsOrigins.Split(',', StringSplitOptions.RemoveEmptyEntries);
        if (origins.Length == 1 && origins[0] == "*")
        {
            policy.AllowAnyOrigin()
                  .AllowAnyHeader()
                  .WithMethods("GET", "POST");
        }
        else
        {
            policy.WithOrigins(origins)
                  .AllowAnyHeader()
                  .WithMethods("GET", "POST")
                  .AllowCredentials();
        }
    });
});

// Database
builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseSqlite(builder.Configuration.GetConnectionString("DefaultConnection")));

// Global database write lock for SQLite concurrency safety
builder.Services.AddSingleton<SemaphoreSlim>(new SemaphoreSlim(1, 1));

// Memory Cache
builder.Services.AddMemoryCache();

// HTTP Client with retry for EastMoney (performance optimization)
builder.Services.AddHttpClient<EastMoneyClient>(client =>
{
    client.DefaultRequestHeaders.Add("User-Agent",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36");
    client.DefaultRequestHeaders.Add("Referer", "https://quote.eastmoney.com/");
    client.Timeout = TimeSpan.FromSeconds(10);
})
.AddTransientHttpErrorPolicy(policy =>
    policy.WaitAndRetryAsync(3, retryAttempt =>
        TimeSpan.FromSeconds(Math.Pow(2, retryAttempt) * 0.5)));

// HTTP Client for FundSyncService
builder.Services.AddHttpClient("FundSync", client =>
{
    client.DefaultRequestHeaders.Add("User-Agent",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36");
    client.DefaultRequestHeaders.Add("Referer", "https://fund.eastmoney.com/");
    client.Timeout = TimeSpan.FromSeconds(15);
})
.AddTransientHttpErrorPolicy(policy =>
    policy.WaitAndRetryAsync(3, retryAttempt =>
        TimeSpan.FromSeconds(Math.Pow(2, retryAttempt) * 0.5)));

// Services
builder.Services.AddSingleton<OnnxPredictionService>();
builder.Services.AddScoped<FundService>();
builder.Services.AddScoped<MarketService>();
builder.Services.AddScoped<PredictionService>();
builder.Services.AddSingleton<FundSyncService>();
builder.Services.AddHostedService(sp => sp.GetRequiredService<FundSyncService>());
builder.Services.AddSingleton<QuoteSyncService>();
builder.Services.AddHostedService(sp => sp.GetRequiredService<QuoteSyncService>());

// Controllers + JSON
builder.Services.AddControllers()
    .AddJsonOptions(options =>
    {
        options.JsonSerializerOptions.PropertyNamingPolicy = null; // Use [JsonPropertyName] attributes
        options.JsonSerializerOptions.DefaultIgnoreCondition =
            JsonIgnoreCondition.Never;
        options.JsonSerializerOptions.Converters.Add(new JsonStringEnumConverter(JsonNamingPolicy.CamelCase));
    });

// Swagger (development only)
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

var app = builder.Build();

// Ensure database exists
using (var scope = app.Services.CreateScope())
{
    var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
    db.Database.EnsureCreated();
    await SeedData.Initialize(db);
}

// Middleware pipeline
if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseResponseCompression();
app.UseMiddleware<RequestLoggingMiddleware>();
app.UseMiddleware<GlobalExceptionMiddleware>();
app.UseCors();
app.MapControllers();

app.Run();
