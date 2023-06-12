package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.CommandHandler;

import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;

public class AppliedAutofixesCommandHandler implements CommandHandler {

    @Override
    public boolean canHandle(String command) {
        return command.equals("applied-autofixes");
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        String autofixIds = System.getProperty("steadybit-applied-autofixes");

        PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8));
        writer.write(RC_OK);
        if (autofixIds != null) {
            writer.println(autofixIds);
        } else {
            writer.println("");
        }
        writer.flush();
    }
}
