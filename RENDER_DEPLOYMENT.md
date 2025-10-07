# Render.com Deployment Configuration

## Web Service Configuration

### Basic Settings
- **Runtime**: Go
- **Build Command**: `./build.sh`
- **Start Command**: `./muambr-api`
- **Environment**: Production

### Environment Variables
Set these in your Render dashboard:

| Variable | Value | Description |
|----------|--------|-------------|
| `PORT` | (auto-set by Render) | Server port (automatically provided) |
| `GIN_MODE` | `release` | Sets Gin to production mode |
| `PYTHON_PATH` | `python3` | Python executable path |

### Auto-Deploy
- **Branch**: `main`
- **Auto-Deploy**: Yes

## Step-by-Step Deployment Guide

### 1. Prepare Your Repository
Ensure these files are in your repository root:
- ✅ `build.sh` (build script)
- ✅ `requirements.txt` (Python dependencies)
- ✅ `go.mod` and `go.sum` (Go dependencies)
- ✅ All source code files

### 2. Create New Web Service on Render

1. Go to [Render Dashboard](https://dashboard.render.com/)
2. Click "New" → "Web Service"
3. Connect your GitHub repository: `neneds/muambr-api`
4. Configure the service:

#### Service Details
```
Name: muambr-api
Environment: Docker (or Go if available)
Region: Choose your preferred region
Branch: main
Root Directory: (leave blank)
```

#### Build & Deploy Settings
```
Build Command: ./build.sh
Start Command: ./muambr-api
```

#### Environment Variables (Advanced → Environment)
```
GIN_MODE=release
PYTHON_PATH=python3
```

### 3. Deploy
1. Click "Create Web Service"
2. Render will automatically:
   - Clone your repository
   - Run the build command
   - Install Python dependencies
   - Build the Go application
   - Start the service

### 4. Access Your API
Once deployed, your API will be available at:
```
https://muambr-api-[random-string].onrender.com
```

## API Endpoints

### Health Check
```bash
curl https://your-app-name.onrender.com/health
```

### Product Comparisons
```bash
# Mercado Livre (Brazil)
curl "https://your-app-name.onrender.com/api/comparisons?name=sony%20xm6&country=BR&currency=BRL"

# KuantoKusta (Portugal)  
curl "https://your-app-name.onrender.com/api/comparisons?name=sony%20xm6&country=PT&currency=EUR"

# Kelkoo (Spain)
curl "https://your-app-name.onrender.com/api/comparisons?name=sony%20xm6&country=ES&currency=EUR"
```

## Build Process Details

### What the build script does:
1. **Install Python Dependencies**: Installs BeautifulSoup4 and lxml for web scraping
2. **Build Go Application**: Compiles the Go source code into a binary
3. **Prepare for Deployment**: Creates the executable `muambr-api`

### Dependencies Installed:
- **Python**: `beautifulsoup4==4.12.2`, `lxml==4.9.3`
- **Go**: All dependencies from `go.mod` (Gin, UUID, etc.)

## Resource Requirements

### Recommended Render Plan:
- **Starter Plan** ($7/month): Should be sufficient for moderate usage
- **Standard Plan** ($25/month): For higher traffic or faster builds

### Resource Usage:
- **CPU**: Moderate (web scraping can be CPU intensive)
- **Memory**: 512MB should be sufficient
- **Storage**: Minimal (stateless application)

## Troubleshooting

### Common Issues:

#### Build Fails
- Check that `build.sh` is executable: `chmod +x build.sh`
- Verify `requirements.txt` is properly formatted
- Check Go mod dependencies: `go mod tidy`

#### Python Import Errors
- Ensure `PYTHON_PATH=python3` environment variable is set
- Verify Python dependencies in `requirements.txt`

#### Port Issues
- Render automatically provides `PORT` environment variable
- Application correctly reads from `os.Getenv("PORT")`

#### Performance Issues
- Consider upgrading Render plan for better performance
- Monitor logs for slow requests
- Implement caching if needed

### Monitoring
- Check Render dashboard for deployment logs
- Monitor service health via `/health` endpoint
- Set up uptime monitoring if needed

## Cost Estimation

### Monthly Costs (approximate):
- **Starter Plan**: $7/month
- **Standard Plan**: $25/month
- **Pro Plan**: $85/month

### Free Tier Limitations:
- Render offers a free tier with limitations:
  - Service spins down after 15 minutes of inactivity
  - 750 hours/month limit
  - Slower cold starts

## Security Considerations

### Current Security:
- CORS enabled for all origins (development setting)
- No authentication implemented
- All endpoints publicly accessible

### Production Recommendations:
1. **Implement Rate Limiting**
2. **Add API Authentication** (API keys, JWT)
3. **Restrict CORS** to specific origins
4. **Add Request Logging**
5. **Monitor for Abuse**

## Additional Features to Consider

### Caching
- Implement Redis for caching extraction results
- Reduce API calls to external sites
- Improve response times

### Database
- Add PostgreSQL for storing popular searches
- Implement user preferences
- Store extraction history

### Monitoring
- Add health check endpoints with detailed status
- Implement structured logging
- Set up error tracking (Sentry, etc.)

---

Your API is now ready for deployment on Render.com! 🚀