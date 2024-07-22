/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.util;

import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;

import java.util.Arrays;
import java.util.HashSet;
import java.util.Set;

class WeakConcurrentSetTest {

    @Test
    void testLocalExpunction() throws Exception {
        final WeakConcurrentSet<Object> set = new WeakConcurrentSet<>(WeakConcurrentSet.Cleaner.INLINE);
        assertThat(set.getCleanerThread()).isNull();
        new SetTestCase(set) {
            @Override
            protected void triggerClean() {
                set.target.expungeStaleEntries();
            }
        }.doTest();
    }

    @Test
    void testExternalThread() throws Exception {
        WeakConcurrentSet<Object> set = new WeakConcurrentSet<>(WeakConcurrentSet.Cleaner.MANUAL);
        assertThat(set.getCleanerThread()).isNull();
        Thread thread = new Thread(set);
        thread.start();
        new SetTestCase(set).doTest();
        thread.interrupt();
        Thread.sleep(200L);
        assertThat(thread.isAlive()).isFalse();
    }

    @Test
    void testInternalThread() throws Exception {
        WeakConcurrentSet<Object> set = new WeakConcurrentSet<>(WeakConcurrentSet.Cleaner.THREAD);
        assertThat(set.getCleanerThread()).isNotNull();
        new SetTestCase(set).doTest();
        set.getCleanerThread().interrupt();
        Thread.sleep(200L);
        assertThat(set.getCleanerThread().isAlive()).isFalse();
    }

    private static class SetTestCase {

        private final WeakConcurrentSet<Object> set;

        SetTestCase(WeakConcurrentSet<Object> set) {
            this.set = set;
        }

        void doTest() throws Exception {
            Object value1 = new Object(), value2 = new Object(), value3 = new Object(), value4 = new Object();
            this.set.add(value1);
            this.set.add(value2);
            this.set.add(value3);
            this.set.add(value4);
            assertThat(this.set.contains(value1)).isTrue();
            assertThat(this.set.contains(value2)).isTrue();
            assertThat(this.set.contains(value3)).isTrue();
            assertThat(this.set.contains(value4)).isTrue();
            Set<Object> values = new HashSet<>(Arrays.asList(value1, value2, value3, value4));
            for (Object value : this.set) {
                assertThat(values.remove(value)).isTrue();
            }
            assertThat(values.isEmpty()).isTrue();
            value1 = value2 = null; // Make eligible for GC
            System.gc();
            Thread.sleep(200L);
            this.triggerClean();
            assertThat(this.set.contains(value3)).isTrue();
            assertThat(this.set.contains(value4)).isTrue();
            assertThat(this.set.approximateSize()).isEqualTo(2);
            assertThat(this.set.target.target.size()).isEqualTo(2);
            assertThat(this.set.remove(value3)).isTrue();
            assertThat(this.set.contains(value3)).isFalse();
            assertThat(this.set.contains(value4)).isTrue();
            assertThat(this.set.approximateSize()).isEqualTo(1);
            assertThat(this.set.target.target.size()).isEqualTo(1);
            this.set.clear();
            assertThat(this.set.contains(value3)).isFalse();
            assertThat(this.set.contains(value4)).isFalse();
            assertThat(this.set.approximateSize()).isEqualTo(0);
            assertThat(this.set.target.target.size()).isEqualTo(0);
            assertThat(this.set.iterator().hasNext()).isFalse();
        }

        protected void triggerClean() {
        }
    }

    @Test
    void testSetContract() {
        final WeakConcurrentSet<Object> set = new WeakConcurrentSet<>(WeakConcurrentSet.Cleaner.INLINE);
        Object obj = new Object();
        assertThat(set.contains(obj)).isFalse();
        assertThat(set.remove(obj)).isFalse();
        assertThat(set.add(obj)).isTrue();
        assertThat(set.add(obj)).isFalse();
        assertThat(set.contains(obj)).isTrue();
        assertThat(set.remove(obj)).isTrue();
        assertThat(set.contains(obj)).isFalse();
        assertThat(set.remove(obj)).isFalse();
    }
}
