/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.http.server.reactive.ServerHttpRequest;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;

import java.net.InetAddress;
import java.net.UnknownHostException;

@Controller
@RequestMapping(path = "/javaagent")
public class
RegisterJavaAgentHandler {
    private final RemoteJvmConnections remoteJvmConnections;

    public RegisterJavaAgentHandler(RemoteJvmConnections remoteJvmConnections) {
        this.remoteJvmConnections = remoteJvmConnections;
    }

    @PutMapping(consumes = MediaType.TEXT_PLAIN_VALUE)
    public ResponseEntity<Void> handle(ServerHttpRequest request, @RequestBody String body) throws UnknownHostException {
        var remoteAddress = request.getRemoteAddress();
        var status = remoteAddress != null ? this.handleInternal(remoteAddress.getAddress(), body) : 500;
        return ResponseEntity.status(status).build();
    }

    int handleInternal(InetAddress remoteAddress, String body) throws UnknownHostException {
        var bodySanitized = body.replaceAll("[\n\r\t]", "_"); //SONAR - Replace pattern-breaking characters
        var bodySplitted = bodySanitized.split("=", 2);
        if (bodySplitted.length == 2) {
            var pid = Integer.parseInt(bodySplitted[0]);
            var jvmRemote = bodySplitted[1];
            var jvmRemoteSplitted = jvmRemote.split(":", 2);
            InetAddress host;
            int port;
            if (jvmRemoteSplitted.length > 1) {
                host = InetAddress.getByName(jvmRemoteSplitted[0]);
                port = Integer.parseInt(jvmRemoteSplitted[1]);
            } else {
                port = Integer.parseInt(jvmRemote);
                host = remoteAddress;
            }

            this.remoteJvmConnections.addConnection(pid, host, port);
            return 200;
        } else {
            return 400;
        }
    }
}
