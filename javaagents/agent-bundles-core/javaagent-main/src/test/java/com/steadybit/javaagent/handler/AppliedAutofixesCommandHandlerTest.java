package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.CommandHandler;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayOutputStream;

class AppliedAutofixesCommandHandlerTest {
    @Test
    void should_return_autofixes() {
        System.setProperty("steadybit-applied-autofixes", "123,456");
        CommandHandler handler = new AppliedAutofixesCommandHandler();

        String response = this.command(handler, "applied-autofixes");
        assertThat(response).isEqualTo("123,456\n");
        System.clearProperty("steadybit-applied-autofixes");
    }

    @Test
    void should_return_empty_autofixes() {
        System.clearProperty("steadybit-applied-autofixes");
        CommandHandler handler = new AppliedAutofixesCommandHandler();

        String response = this.command(handler, "applied-autofixes");
        assertThat(response).isEqualTo("\n");
    }

    private String command(CommandHandler handler, String command) {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        handler.handle(command, "", os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new String(buf, 1, buf.length - 1);
    }
}