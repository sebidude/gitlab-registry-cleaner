FROM scratch
COPY build/gitlab-registry-cleaner /gitlab-registry-cleaner
ENTRYPOINT ["/gitlab-registry-cleaner"]
