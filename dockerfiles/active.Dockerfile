FROM --platform=linux/arm64 debian:13-slim

# Create non-root user
RUN useradd -m -u 10001 appuser

# Copy files and fix ownership
COPY game_server/ /app/
RUN chown -R appuser:appuser /app && \
    find /app -type f -exec chmod +x {} \;

WORKDIR /app
USER appuser

EXPOSE 7777
