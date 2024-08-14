/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.util;

import org.junit.jupiter.api.Test;

import java.util.Arrays;
import java.util.HashSet;
import java.util.Set;

import static org.hamcrest.CoreMatchers.is;
import static org.hamcrest.CoreMatchers.not;
import static org.hamcrest.CoreMatchers.nullValue;
import static org.hamcrest.MatcherAssert.assertThat;

class WeakConcurrentSetTest {

    @Test
    void testLocalExpunction() throws Exception {
        final WeakConcurrentSet<Object> set = new WeakConcurrentSet<>(WeakConcurrentSet.Cleaner.INLINE);
        assertThat(set.getCleanerThread(), nullValue(Thread.class));
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
        assertThat(set.getCleanerThread(), nullValue(Thread.class));
        Thread thread = new Thread(set);
        thread.start();
        new SetTestCase(set).doTest();
        thread.interrupt();
        Thread.sleep(200L);
        assertThat(thread.isAlive(), is(false));
    }

    @Test
    void testInternalThread() throws Exception {
        WeakConcurrentSet<Object> set = new WeakConcurrentSet<>(WeakConcurrentSet.Cleaner.THREAD);
        assertThat(set.getCleanerThread(), not(nullValue(Thread.class)));
        new SetTestCase(set).doTest();
        set.getCleanerThread().interrupt();
        Thread.sleep(200L);
        assertThat(set.getCleanerThread().isAlive(), is(false));
    }

    private class SetTestCase {

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
            assertThat(this.set.contains(value1), is(true));
            assertThat(this.set.contains(value2), is(true));
            assertThat(this.set.contains(value3), is(true));
            assertThat(this.set.contains(value4), is(true));
            Set<Object> values = new HashSet<>(Arrays.asList(value1, value2, value3, value4));
            for (Object value : this.set) {
                assertThat(values.remove(value), is(true));
            }
            assertThat(values.isEmpty(), is(true));
            value1 = value2 = null; // Make eligible for GC
            System.gc();
            Thread.sleep(200L);
            this.triggerClean();
            assertThat(this.set.contains(value3), is(true));
            assertThat(this.set.contains(value4), is(true));
            assertThat(this.set.approximateSize(), is(2));
            assertThat(this.set.target.target.size(), is(2));
            assertThat(this.set.remove(value3), is(true));
            assertThat(this.set.contains(value3), is(false));
            assertThat(this.set.contains(value4), is(true));
            assertThat(this.set.approximateSize(), is(1));
            assertThat(this.set.target.target.size(), is(1));
            this.set.clear();
            assertThat(this.set.contains(value3), is(false));
            assertThat(this.set.contains(value4), is(false));
            assertThat(this.set.approximateSize(), is(0));
            assertThat(this.set.target.target.size(), is(0));
            assertThat(this.set.iterator().hasNext(), is(false));
        }

        protected void triggerClean() {
        }
    }

    @Test
    void testSetContract() throws Exception {
        final WeakConcurrentSet<Object> set = new WeakConcurrentSet<>(WeakConcurrentSet.Cleaner.INLINE);
        Object obj = new Object();
        assertThat(set.contains(obj), is(false));
        assertThat(set.remove(obj), is(false));
        assertThat(set.add(obj), is(true));
        assertThat(set.add(obj), is(false));
        assertThat(set.contains(obj), is(true));
        assertThat(set.remove(obj), is(true));
        assertThat(set.contains(obj), is(false));
        assertThat(set.remove(obj), is(false));
    }
}