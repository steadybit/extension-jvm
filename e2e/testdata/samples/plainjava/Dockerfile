# Plain (non-Spring) Java sample for the attack support matrix.
# Compiled to Java 8 bytecode so a single jar runs on every Java LTS runtime.
ARG BUILDER_IMAGE=eclipse-temurin:17-jdk
ARG RUNTIME_IMAGE=eclipse-temurin:21-jre

FROM ${BUILDER_IMAGE} AS build
ARG COMPILER_RELEASE=8
WORKDIR /src
COPY src src
RUN javac --release ${COMPILER_RELEASE} -d out $(find src -name '*.java') \
    && (cd out && jar --create --file /plain.jar --main-class com.steadybit.matrix.PlainMain .)

FROM ${RUNTIME_IMAGE}
COPY --from=build /plain.jar /plain.jar
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/plain.jar"]
