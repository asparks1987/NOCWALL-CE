# Use the official PHP-Apache image
FROM php:8.2-apache

# Copy composer from official image for dependency install
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

# Install required packages and extensions
RUN apt-get update && apt-get install -y \
    libsqlite3-dev \
    iputils-ping \
    curl \
    unzip \
    ca-certificates \
    && docker-php-ext-install pdo pdo_sqlite \
    && rm -rf /var/lib/apt/lists/*

# Enable Apache modules
RUN a2enmod rewrite

# Set working directory
WORKDIR /var/www/html

# Install PHP dependencies
COPY composer.json /var/www/html/composer.json
RUN composer install --no-dev --prefer-dist --no-interaction --no-progress --optimize-autoloader

# Copy application files
COPY index.php /var/www/html/
COPY assets/ /var/www/html/assets/
COPY buz.mp3 /var/www/html/

# --- Embed Gotify server ---
# Use a known release; override at build time if desired
ARG GOTIFY_VERSION=2.6.0
ARG TARGETOS
ARG TARGETARCH
RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) G_ARCH=amd64 ;; \
      arm64) G_ARCH=arm64 ;; \
      arm)   G_ARCH=arm-7 ;; \
      *) echo "Unsupported TARGETARCH: ${TARGETARCH}"; exit 1 ;; \
    esac; \
    # Try multiple filename patterns and extensions for Gotify release assets
    found=""; \
    for arch in "${G_ARCH}" "x86_64" "aarch64" "armv7" "arm7" "armhf"; do \
      for ext in tar.gz zip; do \
        url="https://github.com/gotify/server/releases/download/v${GOTIFY_VERSION}/gotify-linux-${arch}.${ext}"; \
        echo "Attempting: ${url}"; \
        if curl -fL -o "/tmp/gotify.${ext}" "$url"; then found="$ext:$arch"; break 2; fi; \
      done; \
    done; \
    if [ -z "$found" ]; then echo "Could not download Gotify for arch ${G_ARCH} (v${GOTIFY_VERSION})"; exit 1; fi; \
    case "$found" in \
      tar.gz:*) tar -xzf /tmp/gotify.tar.gz -C /usr/local/bin ;; \
      zip:*)    unzip -o /tmp/gotify.zip -d /usr/local/bin ;; \
    esac; \
    # Normalize binary name to /usr/local/bin/gotify
    if [ -f "/usr/local/bin/gotify-linux-${G_ARCH}" ]; then mv "/usr/local/bin/gotify-linux-${G_ARCH}" /usr/local/bin/gotify; fi; \
    if [ -f "/usr/local/bin/gotify-linux" ]; then mv "/usr/local/bin/gotify-linux" /usr/local/bin/gotify; fi; \
    if [ -f "/usr/local/bin/gotify" ]; then chmod +x /usr/local/bin/gotify; else echo "Gotify binary not found after extraction"; ls -la /usr/local/bin; exit 1; fi; \
    rm -f /tmp/gotify.tar.gz /tmp/gotify.zip; \
    mkdir -p /etc/gotify; \
    printf '%s\n' \
      'server:' \
      '  listenaddr: "0.0.0.0"' \
      '  port: 18080' \
      'database:' \
      '  dialect: "sqlite3"' \
      '  connection: "/var/www/html/cache/gotify/data.db"' \
      > /etc/gotify/config.yml

# Startup script to run Gotify + Apache
COPY start.sh /usr/local/bin/start.sh
RUN chmod +x /usr/local/bin/start.sh

# Ensure permissions
RUN chown -R www-data:www-data /var/www/html \
    && chmod -R 755 /var/www/html

# Expose port 80
EXPOSE 80 18080

# Start Apache
CMD ["/usr/local/bin/start.sh"]
