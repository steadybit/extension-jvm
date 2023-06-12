/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.util;

import java.io.IOException;
import java.io.OutputStream;

/**
 * Stream counting the bytes passed
 */
public class CountingOutputStream extends OutputStream {
    private long count;
    private final OutputStream out;

    public CountingOutputStream(OutputStream out) {
        this.out = out;
    }

    @Override
    public void write(int b) throws IOException {
        this.out.write(b);
        this.count++;
    }

    @Override
    public void write(byte[] b) throws IOException {
        this.out.write(b);
        this.count += b.length;
    }

    @Override
    public void write(byte[] b, int off, int len) throws IOException {
        this.out.write(b, off, len);
        this.count += len;
    }

    @Override
    public void flush() throws IOException {
        this.out.flush();
    }

    @Override
    public void close() throws IOException {
        this.out.close();
    }

    public long getCount() {
        return this.count;
    }

    public void resetCount() {
        this.count = 0;
    }
}