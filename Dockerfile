FROM scratch

COPY /ksp /ksp

ENTRYPOINT [ "/ksp" ]