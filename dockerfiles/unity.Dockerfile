FROM --platform=linux/amd64 debian:13-slim

RUN apt-get update && apt-get install -y \
    libxss1 \
    libgtk-3-0 \
    libxrandr2 \
    libasound2 \
    libpangocairo-1.0-0 \
    libatk1.0-0 \
    libcairo-gobject2 \
    libgdk-pixbuf-xlib-2.0-0 \
    libnss3 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 10001 appuser

# Copy files and fix ownership
COPY game_server/ /app/
RUN chown -R appuser:appuser /app && \
    find /app -type f \( -name "*.x86_64" -o -name "*.exe" \) -exec chmod +x {} \;

WORKDIR /app
USER appuser

EXPOSE 7777
