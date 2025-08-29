FROM --platform=linux/amd64 debian:bullseye-slim

# Install dependencies for Unity
RUN apt-get update && apt-get install -y \
    libgconf-2-4 \
    libxss1 \
    libgtk-3-0 \
    libxrandr2 \
    libasound2 \
    libpangocairo-1.0-0 \
    libatk1.0-0 \
    libcairo-gobject2 \
    libgtk-3-0 \
    libgdk-pixbuf2.0-0 \
    && rm -rf /var/lib/apt/lists/*

# Copy the Unity build
COPY sdd-server-build/ /app/
WORKDIR /app

# Make the binary executable
RUN chmod +x sdd-server-build.x86_64

# Expose the default Mirror port
EXPOSE 7777

# Default command (will be overridden by Go script)
CMD ["./sdd-server-build.x86_64", "-port", "7777"]