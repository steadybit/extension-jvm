/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.log;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.Proxy;
import java.net.URL;
import java.nio.charset.StandardCharsets;

public class RemoteAgentLogger implements Logger {
    private static final String LINE_SEPARATOR = System.getProperty("line.separator");
    private static final String DELIM_STR = "\\{\\}";
    private static LogLevel logLevel;
    private static String pid;
    private static URL remoteLogUrl;
    private static boolean logToSystemOut;
    private final String name;

    static {
        String logLevelString = System.getProperty("STEADYBIT_LOG_LEVEL", System.getenv("STEADYBIT_LOG_LEVEL"));
        RemoteAgentLogger.logLevel = logLevelString != null ? LogLevel.valueOf(logLevelString) : LogLevel.INFO;
        String logToSystemOutEnabledString = System.getProperty("STEADYBIT_LOG_JAVAAGENT_STDOUT", System.getenv("STEADYBIT_LOG_JAVAAGENT_STDOUT"));
        RemoteAgentLogger.logToSystemOut = Boolean.parseBoolean(logToSystemOutEnabledString);
    }

    private RemoteAgentLogger(String name) {
        this.name = name;
    }

    public static Logger getLogger(Class<?> clazz) {
        return new RemoteAgentLogger(clazz.getSimpleName());
    }

    public static void setLevel(LogLevel logLevel) {
        RemoteAgentLogger.logLevel = logLevel;
    }

    public static LogLevel getLevel() {
        return RemoteAgentLogger.logLevel;
    }

    @Override
    public boolean isTraceEnabled() {
        return this.isLogEnabled(LogLevel.TRACE);
    }

    @Override
    public boolean isDebugEnabled() {
        return this.isLogEnabled(LogLevel.DEBUG);
    }

    @Override
    public boolean isInfoEnabled() {
        return this.isLogEnabled(LogLevel.INFO);
    }

    @Override
    public boolean isWarnEnabled() {
        return this.isLogEnabled(LogLevel.WARN);
    }

    @Override
    public boolean isErrorEnabled() {
        return this.isLogEnabled(LogLevel.ERROR);
    }

    public static void init(String pid, String agentHost, String agentPort) {
        RemoteAgentLogger.pid = pid;
        try {
            RemoteAgentLogger.remoteLogUrl = new URL("http://" + agentHost + ":" + agentPort + "/" + "log");
        } catch (MalformedURLException e) {
            System.err.print(String.format("Could not set up remote log url: %s", e.getMessage()));
        }
    }

    public static void setLogToSystem(boolean b) {
        RemoteAgentLogger.logToSystemOut = b;
    }

    private static void appendEscaped(PrintWriter pw, String value) {
        int len = value.length();
        for (int i = 0; i < len; ++i) {
            char ch = value.charAt(i);
            switch (ch) {
            case '\b':
                pw.print("\\b");
                continue;
            case '\t':
                pw.print("\\t");
                continue;
            case '\n':
                pw.print("\\n");
                continue;
            case '\f':
                pw.print("\\f");
                continue;
            case '\r':
                pw.print("\\r");
                continue;
            case '"':
                pw.print("\\\"");
                continue;
            case '/':
                pw.print("\\/");
                continue;
            case '\\':
                pw.print("\\\\");
                continue;
            }
            if (ch >= 0 && ch <= 31 || ch >= 127 && ch <= 159 || ch >= 8192 && ch <= 8447) {
                String ss = Integer.toHexString(ch);
                pw.print("\\u");

                for (int k = 0; k < 4 - ss.length(); ++k) {
                    pw.print('0');
                }
                pw.print(ss.toUpperCase());
            } else {
                pw.print(ch);
            }
        }

    }

    public boolean isLogEnabled(LogLevel level) {
        return RemoteAgentLogger.logLevel.getLevel() >= level.getLevel();
    }

    protected void log(LogLevel level, String msg) {
        if (this.isLogEnabled(level)) {
            this.doLog(level, msg, null);
        }
    }

    protected void log(LogLevel level, String format, Object arg) {
        if (this.isLogEnabled(level)) {
            String msg = String.format(format.replaceAll(DELIM_STR, "%s"), arg);
            this.doLog(level, msg, null);
        }
    }

    protected void log(LogLevel level, String format, Object[] args) {
        if (this.isLogEnabled(level)) {
            String msg = String.format(format.replaceAll(DELIM_STR, "%s"), args);
            this.doLog(level, msg, null);
        }
    }

    protected void log(LogLevel level, String msg, Throwable t) {
        if (this.isLogEnabled(level)) {
            this.doLog(level, msg, t);
        }
    }

    private void doLog(LogLevel level, String msg, Throwable t) {
        if (logToSystemOut) {
            this.doLogToSystemOut(level, msg, t);
        }
        this.doLogToRemoteAgent(level, t, msg);
    }

    private void doLogToSystemOut(LogLevel level, String msg, Throwable t) {
        String msgFormatted = String.format("%-12tT %-5s [%-16s] | %-16s | %s", System.currentTimeMillis(), level.toString(), Thread.currentThread().getName(),
                this.name, msg);
        System.out.print(msgFormatted + LINE_SEPARATOR);
        if (t != null) {
            t.printStackTrace(System.out);
        }
        System.out.flush();
    }

    private void doLogToRemoteAgent(LogLevel level, Throwable t, String msg) {
        if (remoteLogUrl != null) {
            this.sendLog(remoteLogUrl, level, msg, t);
        }
    }

    private byte[] convertToJson(String level, String message, Throwable t) {
        ByteArrayOutputStream baos = new ByteArrayOutputStream();
        PrintWriter pw = new PrintWriter(new OutputStreamWriter(baos, StandardCharsets.UTF_8));
        pw.print("{\"msg\":\"");
        appendEscaped(pw, message);
        pw.print('"');
        pw.print(",\"pid\":\"");
        appendEscaped(pw, pid);
        pw.print('"');
        pw.print(",\"level\":\"");
        appendEscaped(pw, level);
        pw.print('"');
        if (t != null) {
            pw.print(",\"stacktrace\":\"");
            appendEscaped(pw, this.getStackTraceAsString(t));
            pw.print('"');
        }
        pw.print('}');
        pw.close();
        return baos.toByteArray();
    }

    private String getStackTraceAsString(Throwable thrown) {
        StringWriter sw = new StringWriter();
        thrown.printStackTrace(new PrintWriter(sw));
        return sw.toString();
    }

    private void sendLog(URL remoteLogUrl, LogLevel level, String msg, Throwable t) {
        byte[] request = this.convertToJson(level.toString(), msg, t);
        OutputStream outputStream = null;
        boolean successful = false;
        try {
            HttpURLConnection connection = (HttpURLConnection) remoteLogUrl.openConnection(Proxy.NO_PROXY);
            connection.setUseCaches(false);
            connection.setConnectTimeout(500);
            connection.setReadTimeout(1000);
            connection.setRequestMethod("POST");
            connection.setRequestProperty("Content-Type", "application/json");
            connection.setRequestProperty("Content-Length", Integer.toString(request.length));
            connection.setDoOutput(true);
            outputStream = connection.getOutputStream();
            outputStream.write(request);
            outputStream.flush();

            try {
                //to support keep-alive read the response fully
                this.consumeAndClose(connection.getInputStream());
                int status = connection.getResponseCode();
                successful = status >= 200 && status < 300;
                if (!successful) {
                    System.err.println("Javaagent log could not be sent over HTTP to " + remoteLogUrl + ". status:" + status);
                }
            } catch (IOException e) {
                //to support keep-alive read the response fully
                this.consumeAndClose(connection.getErrorStream());
            }

        } catch (IOException e) {
            System.err.println("Javaagent log could not be sent over HTTP to " + remoteLogUrl + ": " + e.getMessage());
        } finally {
            if (outputStream != null) {
                try {
                    outputStream.close();
                } catch (IOException e) {
                    //ignore - we can't do a thing about it.
                }
            }
        }
        if (!successful && !RemoteAgentLogger.logToSystemOut) {
            // Do log to SdtOut if not already done.
            this.doLogToSystemOut(level, msg, t);
        }
    }

    private void consumeAndClose(InputStream s) {
        if (s == null) {
            return;
        }

        byte[] buffer = new byte[1024];
        try {
            while (s.read(buffer) != -1) {
                //just read the entire damn thing
            }
        } catch (IOException e) {
            //ignore - we can't do a thing about it.
        } finally {
            try {
                s.close();
            } catch (IOException e) {
                //ignore - we can't do a thing about it.
            }
        }
    }

    @Override
    public void trace(String msg) {
        this.log(LogLevel.TRACE, msg);
    }

    @Override
    public void trace(String format, Object arg) {
        this.log(LogLevel.TRACE, format, arg);
    }

    @Override
    public void trace(String format, Object[] argArray) {
        this.log(LogLevel.TRACE, format, argArray);
    }

    @Override
    public void trace(String msg, Throwable t) {
        this.log(LogLevel.TRACE, msg, t);
    }

    @Override
    public void debug(String msg) {
        this.log(LogLevel.DEBUG, msg);
    }

    @Override
    public void debug(String format, Object arg) {
        this.log(LogLevel.DEBUG, format, arg);
    }

    @Override
    public void debug(String format, Object[] argArray) {
        this.log(LogLevel.DEBUG, format, argArray);
    }

    @Override
    public void debug(String msg, Throwable t) {
        this.log(LogLevel.DEBUG, msg, t);
    }

    @Override
    public void info(String msg) {
        this.log(LogLevel.INFO, msg);
    }

    @Override
    public void info(String format, Object arg) {
        this.log(LogLevel.INFO, format, arg);
    }

    @Override
    public void info(String format, Object[] argArray) {
        this.log(LogLevel.INFO, format, argArray);
    }

    @Override
    public void info(String msg, Throwable t) {
        this.log(LogLevel.INFO, msg, t);
    }

    @Override
    public void warn(String msg) {
        this.log(LogLevel.WARN, msg);
    }

    @Override
    public void warn(String format, Object arg) {
        this.log(LogLevel.WARN, format, arg);
    }

    @Override
    public void warn(String format, Object[] argArray) {
        this.log(LogLevel.WARN, format, argArray);
    }

    @Override
    public void warn(String msg, Throwable t) {
        this.log(LogLevel.WARN, msg, t);
    }

    @Override
    public void error(String msg) {
        this.log(LogLevel.ERROR, msg);
    }

    @Override
    public void error(String format, Object arg) {
        this.log(LogLevel.ERROR, format, arg);
    }

    @Override
    public void error(String format, Object[] argArray) {
        this.log(LogLevel.ERROR, format, argArray);
    }

    @Override
    public void error(String msg, Throwable t) {
        this.log(LogLevel.ERROR, msg, t);
    }

}
