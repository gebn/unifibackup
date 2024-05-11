FROM scratch
ARG TARGETPLATFORM
COPY docker/$TARGETPLATFORM/unifibackup /
USER nobody
EXPOSE 9184
ENTRYPOINT ["/unifibackup"]
