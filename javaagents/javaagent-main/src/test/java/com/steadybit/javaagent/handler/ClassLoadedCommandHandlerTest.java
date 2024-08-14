/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.CommandHandler;
import com.steadybit.javaagent.LoadedClassesCache;
import org.junit.jupiter.api.Test;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStreamReader;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class ClassLoadedCommandHandlerTest {
    private final LoadedClassesCache classesCache = mock(LoadedClassesCache.class);
    private final CommandHandler handler = new ClassLoadedCommandHandler(this.classesCache);

    @Test
    void should_return_true() throws IOException {
        //given
        when(this.classesCache.isClassLoaded("java.lang.Object")).thenReturn(true);

        //when
        String result = this.command("class-loaded", "java.lang.Object");

        //then
        assertThat(result).isEqualTo("true");
    }

    @Test
    void should_return_false() throws IOException {
        //given
        when(this.classesCache.isClassLoaded("java.lang.Object")).thenReturn(false);

        //when
        String result = this.command("class-loaded", "java.lang.Object");

        //then
        assertThat(result).isEqualTo("false");
    }

    private String command(String command, String argument) throws IOException {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        this.handler.handle(command, argument, os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new BufferedReader(new InputStreamReader(new ByteArrayInputStream(buf, 1, buf.length - 1))).readLine();
    }
}