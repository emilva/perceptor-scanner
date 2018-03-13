FROM alpine:3.6 as packager
ADD jdk-9-ea+181_linux-x64-musl_bin.tar /opt/
# Build our own jdk9 distro with only these modules, as first stage
RUN /opt/jdk-9/bin/jlink \
    --module-path /opt/jdk-9/jmods \
    --verbose \
    --add-modules java.base,java.logging,java.xml,jdk.unsupported \
    --compress 2 \
    --no-header-files \
    --output /opt/jdk-9-minimal

FROM scratch

COPY --from=packager /opt/jdk-9-minimal /opt/jdk-9-minimal

ENV JAVA_HOME=/opt/jdk-9-minimal
ENV PATH="$PATH:$JAVA_HOME/bin"

# Bring in other dependencies
COPY ./perceptor-scanner ./perceptor-scanner

# Run perceptor-scanner
CMD ["./perceptor-scanner"]
