# 🚀 Render.com Deployment - Quick Setup Guide

## Render.com Configuration

### Build Command
```bash
./build.sh
```

### Start Command  
```bash
./muambr-api
```

### Environment Variables
```
GIN_MODE=release
PYTHON_PATH=python3
```

## Files Added for Deployment

✅ **build.sh** - Build script that installs Python deps and builds Go app  
✅ **requirements.txt** - Python dependencies (beautifulsoup4, lxml)  
✅ **Dockerfile** - Alternative Docker deployment  
✅ **RENDER_DEPLOYMENT.md** - Complete deployment guide  

## Code Changes Made

### 1. Port Configuration
- Updated `main.go` to use `PORT` environment variable
- Falls back to `:8080` for local development

### 2. Python Path Configuration  
- Updated all extractors to use configurable Python path
- Uses `PYTHON_PATH` environment variable
- Falls back to `python3` in production

### 3. Dependencies
- Added `requirements.txt` for Python packages
- Made build script executable

## Quick Deploy Steps

1. **Push to GitHub** (if not already done)
2. **Create Render Web Service**:
   - Connect your `neneds/muambr-api` repository
   - Set build command: `./build.sh`  
   - Set start command: `./muambr-api`
3. **Add Environment Variables**:
   - `GIN_MODE=release`
   - `PYTHON_PATH=python3`
4. **Deploy!** 🎉

## Test Your Deployment

Once deployed, test with:
```bash
# Health check
curl https://your-app.onrender.com/health

# API test (Brazil)
curl "https://your-app.onrender.com/api/comparisons?name=sony%20xm6&country=BR"
```

## Cost: ~$7/month (Starter Plan)

Your API is now production-ready for Render.com! 🌟