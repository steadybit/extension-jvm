/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent;

import org.json.JSONObject;
import org.json.JSONTokener;

import java.io.IOException;
import java.io.InputStream;
import java.io.PrintWriter;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.Proxy;
import java.net.URL;

public class HttpAttackClient {
    private final URL baseUrl;

    public HttpAttackClient(String monitorUrl) {
        try {
            this.baseUrl = new URL(monitorUrl);
        } catch (MalformedURLException e) {
            throw new HttpAttackClientException(e);
        }
    }

    public boolean checkAttackStillRunning() {
        try {
            HttpURLConnection conn = this.openConnection("", "GET");
            int status = conn.getResponseCode();
            conn.disconnect();
            return status >= 200 && status < 300;
        } catch (IOException e) {
            return false;
        }
    }

    public JSONObject fetchAttackConfig() {
        try {
            HttpURLConnection conn = this.openConnection("", "GET");
            conn.setDoInput(true);
            conn.setRequestProperty("Accept", "application/json");
            try {
                this.assertStatusOK(conn.getResponseCode());
                try (InputStream is = conn.getInputStream()) {
                    Object config = new JSONTokener(is).nextValue();
                    if (config instanceof JSONObject) {
                        return (JSONObject) config;
                    } else {
                        throw new HttpAttackClientException("Expected " + JSONObject.class + ", but got " + config.getClass());
                    }
                }
            } finally {
                conn.disconnect();
            }
        } catch (IOException e) {
            throw new HttpAttackClientException(e);
        }
    }

    public void attackStarted() {
        try {
            HttpURLConnection conn = this.openConnection("/started", "POST");
            try {
                this.assertStatusOK(conn.getResponseCode());
            } finally {
                conn.disconnect();
            }
        } catch (IOException e) {
            throw new HttpAttackClientException(e);
        }
    }

    public void attackStopped() {
        try {
            HttpURLConnection conn = this.openConnection("/started", "POST");
            try {
                this.assertStatusOK(conn.getResponseCode());
            } finally {
                conn.disconnect();
            }
        } catch (IOException e) {
            throw new HttpAttackClientException(e);
        }
    }

    public void attackFailed(Throwable t) {
        try {
            HttpURLConnection conn = this.openConnection("/failed", "POST");
            conn.setDoOutput(true);
            try {
                try (PrintWriter pw = new PrintWriter(conn.getOutputStream())) {
                    pw.println(t.getMessage());
                    t.printStackTrace(pw);
                }
                this.assertStatusOK(conn.getResponseCode());
            } finally {
                conn.disconnect();
            }
        } catch (IOException e) {
            throw new HttpAttackClientException(e);
        }
    }

    private HttpURLConnection openConnection(String path, String method) throws IOException {
        HttpURLConnection conn = (HttpURLConnection) new URL(this.baseUrl + path).openConnection(Proxy.NO_PROXY);
        conn.setUseCaches(false);
        conn.setConnectTimeout(500);
        conn.setReadTimeout(1000);
        conn.setRequestMethod(method);
        conn.setDoOutput(false);
        return conn;
    }

    private void assertStatusOK(int status) {
        if (status >= 400 && status <= 599) {
            throw new HttpAttackClientException("Unexpected http status code " + status);
        }
    }
}
