@echo off
echo Showing SuperPlane logs (press Ctrl+C to exit)...
echo.
docker compose -f docker-compose.dev.yml logs -f
