ARG SOURCE_DATE_EPOCH=0

# Step 1: Build the application
# --------------------------------------------------------
# Use the official Go image for building the application
FROM debian:bookworm-20250811-slim AS base
ARG SOURCE_DATE_EPOCH
# Install basic OS utilities
RUN --mount=type=bind,source=./repro-sources-list.sh,target=/usr/local/bin/repro-sources-list.sh \
    repro-sources-list.sh && \
    apt-get update && \
    apt-get install --yes --no-install-recommends ca-certificates tzdata sudo

RUN mkdir -p /app/dl && \
    mkdir -p /app/config && \
    mkdir -p /app/logs && \
    mkdir -p /app/stats && \
    mkdir -p /app/data && \
    mkdir -p /app/bin

# Load the binary from the source
COPY bin/hammer* /app/dl/

# Rename the binary for the architecture
RUN ARCH=$(uname -m) && \
    if [ "${ARCH}" = "x86_64" ]; then \
        ARCH="amd64"; \
    elif [ "${ARCH}" = "aarch64" ]; then \
        ARCH="arm64"; \
    fi && \
    cd /app/dl && \
    sha256sum -c hammer-linux-${ARCH}.sha256 && \
    mv /app/dl/hammer-linux-${ARCH} /app/bin/hammer && \
    chmod +x /app/bin/hammer && \
    rm -rf /app/dl/

# Create a non-root user and group same as hedera user (2000:2000)
ENV USER_NAME=hammer
ENV USER_ID=2000
ENV GROUP_ID=2000

RUN groupadd \
    --gid "${GROUP_ID}" \
    --system \
    "${USER_NAME}"

RUN useradd \
    --system \
    --home-dir "/app" \
    --shell "/bin/bash" \
    --no-create-home \
    --uid "${USER_ID}" \
    --gid "${GROUP_ID}" \
    "${USER_NAME}"

RUN echo "hammer ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/hammer && \
    chmod 0440 /etc/sudoers.d/hammer

RUN chown -R hammer:hammer /app/config /app/logs /app/stats /app/data && \
    chmod -R 755 /app/config /app/logs /app/stats /app/data

# Remove Unneeded Utilities
RUN --mount=type=bind,source=./repro-sources-list.sh,target=/usr/local/bin/repro-sources-list.sh \
    repro-sources-list.sh && \
    apt-get autoremove --yes && \
    apt-get autoclean --yes && \
    apt-get clean all --yes && \
    rm -rf /var/log/ && \
    rm -rf /var/cache/


########################################
####    Deterministic Build Hack    ####
########################################

# === Workarounds below will not be needed when https://github.com/moby/buildkit/pull/4057 is merged ===
# NOTE: PR #4057 has been merged but will not be available until the v0.13.x series of releases.
# Limit the timestamp upper bound to SOURCE_DATE_EPOCH.
# Workaround for https://github.com/moby/buildkit/issues/3180
RUN find $( ls / | grep -E -v "^(dev|mnt|proc|sys)$" ) \
  -newermt "@${SOURCE_DATE_EPOCH}" -writable -xdev \
  | xargs touch --date="@${SOURCE_DATE_EPOCH}" --no-dereference

# Step 2: Create a minimal image
# --------------------------------------------------------
# Use a minimal base image for the final container
FROM scratch

COPY --from=base / /

# Set the working directory inside the container
WORKDIR /app

# Set the user to the non-root user
USER hammer:hammer

# Define volumes for data and config
VOLUME ["/app/data", "/app/config", "/app/logs", "/app/stats"]

EXPOSE 6060

ENTRYPOINT ["/app/bin/hammer"]
