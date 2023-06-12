/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.util;

import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayOutputStream;
import java.io.IOException;

class CountingOutputStreamTest {

    @Test
    void should_count_written_bytes() throws IOException {
        try (ByteArrayOutputStream bos = new ByteArrayOutputStream(); CountingOutputStream counting = new CountingOutputStream(bos)) {
            assertThat(counting.getCount()).isEqualTo(0);

            counting.write(1);
            counting.write(new byte[] {2,3});
            counting.write(new byte[] {1,2,3,4,5}, 3,2);

            assertThat(bos.toByteArray()).containsExactly(1,2,3,4,5);
            assertThat(counting.getCount()).isEqualTo(5);

            counting.resetCount();
            assertThat(counting.getCount()).isEqualTo(0);
        }
    }
}