# Base container image
FROM debian:stable

# Copy over the source files into the /src directory
COPY . /src

# Update the container image and build the binary
RUN apt-get update \
    && apt-get upgrade -qq -y \
    && apt-get install -qq -y ca-certificates golang \
    && cd /src/cmd \
    && go build -ldflags '-s -w' -o leaf-bin \
    && apt-get remove -qq -y golang \
    && apt-get autoremove -y
