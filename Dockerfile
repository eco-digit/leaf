# Base container image
FROM debian:stable

# Copy code to container
COPY . /src

# Update the container image and build the binary
RUN cd /src \
    && apt-get update \
    && apt-get upgrade -qq -y \
    && apt-get install -qq -y ca-certificates golang \
    && go build -ldflags '-s -w' -o leaf-bin cmd/leaf/main.go \
    && apt-get remove -qq -y golang \
    && apt-get autoremove -y
