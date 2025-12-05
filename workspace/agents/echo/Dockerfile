# Echo Agent Image for E2E Testing
# This is a simple test agent that echoes the content of task.md
FROM alpine:3.21

# Install minimal dependencies
RUN apk add --no-cache bash

# Create non-root user for security
RUN adduser -D -s /bin/bash agent
USER agent
WORKDIR /workspace

# Default entrypoint - simply echo the content of task.md
# This is useful for e2e testing to verify the task file is correctly mounted
ENTRYPOINT ["sh", "-c", "echo '=== Task Content ===' && cat /workspace/task.md && echo '' && echo '=== Task Completed ==='"]
