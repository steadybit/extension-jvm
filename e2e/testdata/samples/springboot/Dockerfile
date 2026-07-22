# Parameterized Spring Boot sample for the attack support matrix.
# Build args select the Spring Boot version (build stage) and Java runtime (final stage),
# so one source tree produces every (Java LTS x Spring Boot) image in the matrix.
ARG BUILDER_IMAGE=maven:3.9.11-eclipse-temurin-17
ARG RUNTIME_IMAGE=eclipse-temurin:21-jre

FROM ${BUILDER_IMAGE} AS build
ARG BOOT_VERSION=3.5.16
ARG COMPILER_RELEASE=17
# RestClient only exists on Spring Framework 6.1+ (Boot 3.2+); pass empty for Boot 2.7.
ARG MVN_PROFILES=-Pwith-restclient
WORKDIR /src
COPY pom.xml .
COPY src src
RUN --mount=type=cache,target=/root/.m2 \
    mvn -B -e -q \
      -Dboot.version=${BOOT_VERSION} \
      -Dmaven.compiler.source=${COMPILER_RELEASE} \
      -Dmaven.compiler.target=${COMPILER_RELEASE} \
      ${MVN_PROFILES} \
      -DskipTests package

FROM ${RUNTIME_IMAGE}
COPY --from=build /src/target/app.jar /app.jar
ENV DOWNSTREAM_URL=http://localhost:8080/mvc
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app.jar"]
