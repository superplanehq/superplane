@echo off
echo run semaphore integration test
echo.
docker compose -f docker-compose.dev.yml exec app go test -v ./pkg/integrations/semaphore/... -run ListPipelines
if %errorlevel% neq 0 (
    echo.
    echo tests failed
    pause
    exit /b 1
)
echo.
echo all tests passed
echo.
pause
