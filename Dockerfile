FROM scratch
ARG TARGETPLATFORM
COPY docker/$TARGETPLATFORM/unifibackup /
USER nobody
ENTRYPOINT ["/unifibackup"]
