#!/bin/bash

# Stop script on any error
set -e

# --- Configuration ---
IMAGE_NAME="liven-one-backend" # Replace with your desired Docker image name
CONTAINER_NAME="liven-one-backend-container" # Replace with your desired container name
ENV_FILE=".env" # Your environment file
APP_PORT_HOST="8080" # Port on your host machine
APP_PORT_CONTAINER="8080" # Port your Go app listens on INSIDE the container (should match EXPOSE in Dockerfile and PORT env var)
APP_ENV="development" # Port your Go app listens on INSIDE the container (should match EXPOSE in Dockerfile and PORT env var)

# --- Script Logic ---

echo "--- Starting Local Deployment Script ---"

# 1. Check if .env file exists and source variables
if [ ! -f "$ENV_FILE" ]; then
    echo "Error: Environment file '$ENV_FILE' not found."
    echo "Please create it with required configurations (DATABASE_URI, JWT_SECRET)."
    exit 1
fi

echo "Loading environment variables from '$ENV_FILE'..."

# Initialize shell variables
DB_URI_FROM_ENV=""
JWT_SECRET_FROM_ENV=""
PORT_FROM_ENV=""

# Read .env file line by line, and extract values for specific keys
# This loop handles simple KEY=VALUE pairs and ignores comments and empty lines.
# It assumes values do not contain newlines.
while IFS='=' read -r key value || [ -n "$key" ]; do
    # Remove potential leading/trailing whitespace and carriage returns
    key=$(echo "$key" | awk '{$1=$1};1' | tr -d '\r')
    value=$(echo "$value" | awk '{$1=$1};1' | tr -d '\r') # Preserves internal spaces in value

    # Skip comments and empty lines
    if [[ "$key" == \#* ]] || [[ -z "$key" ]]; then
        continue
    fi

    # Assign to specific shell variables
    # Ensure to only take the part of the value before any potential inline comment if your .env has them
    # For simplicity, this example assumes values don't have inline '#' comments after them.
    # If they do, a more complex sed/awk for 'value' would be needed.

    # Strip potential quotes from value (optional, if your .env values might be quoted)
    # value="${value%\"}"
    # value="${value#\"}"

    if [[ "$key" == "DATABASE_URI" ]]; then
        DB_URI_FROM_ENV="$value"
    elif [[ "$key" == "JWT_SECRET" ]]; then
        JWT_SECRET_FROM_ENV="$value"
    elif [[ "$key" == "PORT" ]]; then
        PORT_FROM_ENV="$value"
    fi
done < "$ENV_FILE"

# Validate that essential variables were loaded
if [ -z "$DB_URI_FROM_ENV" ]; then
    echo "Error: DATABASE_URI not found or empty in $ENV_FILE."
    exit 1
fi
if [ -z "$JWT_SECRET_FROM_ENV" ]; then
    echo "Error: JWT_SECRET not found or empty in $ENV_FILE."
    exit 1
fi

# Determine the container port: use from .env if set, otherwise use script default
APP_PORT_CONTAINER="$APP_PORT_HOST"
if [ -n "$PORT_FROM_ENV" ]; then
    APP_PORT_CONTAINER="$PORT_FROM_ENV"
    echo "Using PORT=$APP_PORT_CONTAINER for the container (from $ENV_FILE)."
else
    echo "Using default PORT=$APP_PORT_CONTAINER for the container."
fi


# 2. Build the Docker image
echo "Building Docker image '$IMAGE_NAME'..."
docker build -t "$IMAGE_NAME" .
echo "Docker image '$IMAGE_NAME' built successfully."

# 3. Stop and remove any existing container with the same name
if [ "$(docker ps -q -f name="$CONTAINER_NAME")" ]; then
    echo "Stopping existing container '$CONTAINER_NAME'..."
    docker stop "$CONTAINER_NAME"
fi
if [ "$(docker ps -aq -f name="$CONTAINER_NAME")" ]; then
    echo "Removing existing container '$CONTAINER_NAME'..."
    docker rm "$CONTAINER_NAME"
fi

# 4. Run the new Docker container
echo "Running new container '$CONTAINER_NAME' from image '$IMAGE_NAME'..."
echo "Environment Setting: '$APP_ENV'"
echo "Host Port: $APP_PORT_HOST, Container Port: $APP_PORT_CONTAINER"
echo "DATABASE_URI will be set from $ENV_FILE"
echo "JWT_SECRET will be set from $ENV_FILE"


docker run -d \
    --name "$CONTAINER_NAME" \
    -p "$APP_PORT_HOST":"$APP_PORT_CONTAINER" \
    -e "PORT=$APP_PORT_CONTAINER" \
    -e "APP_ENV=$APP_ENV" \
    -e "DATABASE_URI=$DB_URI_FROM_ENV" \
    -e "JWT_SECRET=$JWT_SECRET_FROM_ENV" \
    "$IMAGE_NAME"
    # Add other -e flags as needed for other variables from your .env

    # Optional: Volume mount for SQLite persistence
    # Create a 'data' directory first: mkdir -p data
    # Then add this line to docker run:
    # -v "$(pwd)/data:/app/data" \
    # And ensure your DATABASE_URI in .env (and thus $DB_URI_FROM_ENV)
    # points to something like /app/data/your_app.db if you want the DB file inside /app/data
    # Example if DATABASE_URI in .env is /app/data/livenone_local.db:
    # docker run -d \
    #     --name "$CONTAINER_NAME" \
    #     -p "$APP_PORT_HOST":"$APP_PORT_CONTAINER" \
    #     -e "PORT=$APP_PORT_CONTAINER" \
    #     -e "DATABASE_URI=$DB_URI_FROM_ENV" \ # This would be /app/data/livenone_local.db
    #     -e "JWT_SECRET=$JWT_SECRET_FROM_ENV" \
    #     -v "$(pwd)/data:/app/data" \
    #     "$IMAGE_NAME"

echo ""
echo "Container '$CONTAINER_NAME' started."
echo "Access your application at http://localhost:$APP_PORT_HOST"
echo "To see logs: docker logs -f $CONTAINER_NAME"
echo "To stop: docker stop $CONTAINER_NAME"
echo "--- Local Deployment Script Finished ---"
