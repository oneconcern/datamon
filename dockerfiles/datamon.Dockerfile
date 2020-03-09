FROM reg.onec.co/datamon-alpine-base:latest as base

#Build the dist image
FROM scratch
COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip
ENTRYPOINT [ "/usr/bin/datamon" ]
CMD ["--help"]
