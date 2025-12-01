# Gemini CLI Agent Image
FROM node:24-slim

# Install git and other common tools needed for repository operations
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Gemini CLI globally
RUN npm install -g @google/gemini-cli

# Create non-root user for security
RUN useradd -m -s /bin/bash gemini
USER gemini
WORKDIR /home/gemini

# Default entrypoint
ENTRYPOINT ["gemini"]
