FROM alpine as builder

RUN apk add shadow ca-certificates
RUN useradd -u 10001 cleaner

FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY build/gitlab-registry-cleaner /gitlab-registry-cleaner

USER cleaner
ENTRYPOINT ["/gitlab-registry-cleaner"]
