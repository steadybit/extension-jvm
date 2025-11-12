/*
 * Copyright 2025 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import java.lang.invoke.MethodHandle;
import java.lang.invoke.MethodHandles;
import java.lang.invoke.MethodType;
import java.net.URI;
import java.util.List;
import java.util.function.Predicate;

public class HttpMatcher {
    private final List<String> httpMethods;
    private final String hostAddress;
    private final Predicate<String> pathPattern;

    public HttpMatcher(List<String> httpMethods, String hostAddress, String urlPath) {
        this.httpMethods = httpMethods;
        this.hostAddress = hostAddress;
        this.pathPattern = this.parsePattern(urlPath);
    }

    private Predicate<String> parsePattern(String urlPath) {
        if (urlPath == null || urlPath.isEmpty() || "/**".equals(urlPath)) {
            return path -> true;
        }

        // Yes this is pretty ugly, but we need the class to be loaded via the context classloader, as this will have
        // the spring classes loaded, so we do the reflection dance here.
        // this is basically:
        // ```
        // PathPatternParser parser = new PathPatternParser();
        // parser.setCaseSensitive(false);
        // PathPattern pattern = parser.parse(urlPath);
        // return url -> return pattern.matches(PathContainer.parsePath(url));
        // ```
        try {
            MethodHandles.Lookup lu = MethodHandles.publicLookup();
            ClassLoader classLoader = Thread.currentThread().getContextClassLoader();

            Class<?> parserClazz = classLoader.loadClass("org.springframework.web.util.pattern.PathPatternParser");
            Object parser = lu.findConstructor(parserClazz, MethodType.methodType(void.class)).invoke();
            lu.findVirtual(parserClazz, "setCaseSensitive", MethodType.methodType(void.class, boolean.class)).invoke(parser, false);

            Class<?> patternClass = classLoader.loadClass("org.springframework.web.util.pattern.PathPattern");
            Object pattern = lu.findVirtual(parserClazz, "parse", MethodType.methodType(patternClass, String.class)).invoke(parser, urlPath);

            Class<?> containerClass = classLoader.loadClass("org.springframework.http.server.PathContainer");
            MethodHandle parsePath = lu.findStatic(containerClass, "parsePath", MethodType.methodType(containerClass, String.class));
            MethodHandle matchers = lu.findVirtual(patternClass, "matches", MethodType.methodType(boolean.class, containerClass));

            return url -> {
                try {
                    return (boolean) matchers.invoke(pattern, parsePath.invoke(url));
                } catch (Error | RuntimeException e) {
                    throw e;
                } catch (Throwable e) {
                    return false;
                }
            };
        } catch (Error | RuntimeException e) {
            throw e;
        } catch (Throwable e) {
            throw new RuntimeException(e);
        }
    }

    public boolean test(String httpMethod, URI uri) {
        int port = uri.getPort();
        if (port == -1) {
            port = uri.getScheme().equalsIgnoreCase("https") ? 443 : 80;
        }
        String host = uri.getHost() + ":" + port;

        if (!"*".equals(this.hostAddress) && !this.hostAddress.equalsIgnoreCase(host)) {
            return false;
        }

        String path = uri.getPath();
        if (!this.pathPattern.test(path)) {
            return false;
        }

        if (this.httpMethods.isEmpty() || httpMethod == null) {
            return true;
        }
        for (String method : this.httpMethods) {
            if (method.equalsIgnoreCase(httpMethod)) {
                return true;
            }
        }
        return false;
    }
}
