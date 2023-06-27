/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.instrumentation;

public class InstrumentationPlugin {
    public static final InstrumentationPlugin NOOP = new InstrumentationPlugin();
    private int registration = -1;

    void setRegistration(int registration) {
        this.registration = registration;
    }

    public int getRegistration() {
        return this.registration;
    }

    public Object exec(int code) {
        return null;
    }

    public Object exec(int code, Object arg1) {
        return null;
    }

    public Object exec(int code, Object arg1, Object arg2) {
        return null;
    }

    public Object exec(int code, Object arg1, Object arg2, Object arg3) {
        return null;
    }

    public Object exec(int code, Object arg1, Object arg2, Object arg3, Object arg4) {
        return null;
    }

    public Object exec(int code, Object arg1, Object arg2, Object arg3, Object arg4, Object arg5) {
        return null;
    }

    public Object exec(int code, Object arg1, Object arg2, Object arg3, Object arg4, Object arg5, Object arg6) {
        return null;
    }

    public Object exec(int code, Object arg1, Object arg2, Object arg3, Object arg4, Object arg5, Object arg6, Object arg7) {
        return null;
    }

    public Object exec(int code, Object arg1, Object arg2, Object arg3, Object arg4, Object arg5, Object arg6, Object arg7, Object arg8) {
        return null;
    }

}
