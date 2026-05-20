@echo off
cd /d "%~dp0"
set GOOGLE_CLOUD_PROJECT=project-93aa7ef8-3fc1-4aa6-868
set GOOGLE_CLOUD_LOCATION=global
echo ============================================
echo GOOGLE_CLOUD_PROJECT=%GOOGLE_CLOUD_PROJECT%
echo GOOGLE_CLOUD_LOCATION=%GOOGLE_CLOUD_LOCATION%
echo ============================================
go run main.go
