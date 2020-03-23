ARG VERSION=20200307
FROM gcr.io/onec-co/datamon-sidecar-base:$VERSION

USER developer
RUN touch ~/.zshrc

ENTRYPOINT [ "/bin/tini", "--"]
CMD ["zsh"]
