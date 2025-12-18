FROM --platform=linux/amd64 debian:13-slim

RUN apt-get update && apt-get install -y \
    libgconf-2-4 \
    libxss1 \
    libgtk-3-0 \
    libxrandr2 \
    libasound2 \
    libpangocairo-1.0-0 \
    libatk1.0-0 \
    libcairo-gobject2 \
    libgdk-pixbuf2.0-0 \
    libnss3 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 10001 unity

# Copy files and fix ownership
COPY game_server/ /app/
RUN chown -R unity:unity /app

WORKDIR /app
USER unity

EXPOSE 7777

CMD ["echo", "Server binary will be set by indiekku"]
