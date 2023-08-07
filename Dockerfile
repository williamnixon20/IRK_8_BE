# Use the official Go image as the base image
FROM golang:1.16

# Set the working directory inside the container
WORKDIR /app

# Copy the contents of your Go project to the container's working directory
COPY . .

# Build the Go application inside the container
RUN go build -o app

# Expose the port your Go application listens on (if applicable)
EXPOSE 8080
# Command to run your Go application when the container starts
CMD ["./app"]
