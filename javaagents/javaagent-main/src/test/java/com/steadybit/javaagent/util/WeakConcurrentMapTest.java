/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.util;

import org.junit.jupiter.api.Test;

import java.lang.ref.WeakReference;
import java.util.HashMap;
import java.util.Map;

import static org.hamcrest.CoreMatchers.is;
import static org.hamcrest.CoreMatchers.not;
import static org.hamcrest.CoreMatchers.nullValue;
import static org.hamcrest.MatcherAssert.assertThat;

class WeakConcurrentMapTest {

    @Test
    void testLocalExpunction() throws Exception {
        final WeakConcurrentMap.WithInlinedExpunction<Object, Object> map = new WeakConcurrentMap.WithInlinedExpunction<>();
        assertThat(map.getCleanerThread(), nullValue(Thread.class));
        new MapTestCase(map) {
            @Override
            protected void triggerClean() {
                map.expungeStaleEntries();
            }
        }.doTest();
    }

    @Test
    void testExternalThread() throws Exception {
        WeakConcurrentMap<Object, Object> map = new WeakConcurrentMap<>(false);
        assertThat(map.getCleanerThread(), nullValue(Thread.class));
        Thread thread = new Thread(map);
        thread.start();
        new MapTestCase(map).doTest();
        thread.interrupt();
        Thread.sleep(200L);
        assertThat(thread.isAlive(), is(false));
    }

    @Test
    void testInternalThread() throws Exception {
        WeakConcurrentMap<Object, Object> map = new WeakConcurrentMap<>(true);
        assertThat(map.getCleanerThread(), not(nullValue(Thread.class)));
        new MapTestCase(map).doTest();
        map.getCleanerThread().interrupt();
        Thread.sleep(200L);
        assertThat(map.getCleanerThread().isAlive(), is(false));
    }

    static class KeyEqualToWeakRefOfItself {
        @Override
        public boolean equals(Object obj) {
            if (obj instanceof WeakReference<?>) {
                return this.equals(((WeakReference<?>) obj).get());
            }
            return super.equals(obj);
        }
    }

    static class CheapUnloadableWeakConcurrentMap extends AbstractWeakConcurrentMap<KeyEqualToWeakRefOfItself, Object, Object> {

        @Override
        protected Object getLookupKey(KeyEqualToWeakRefOfItself key) {
            return key;
        }

        @Override
        protected void resetLookupKey(Object lookupKey) {
        }
    }

    @Test
    void testKeyWithWeakRefEquals() {
        CheapUnloadableWeakConcurrentMap map = new CheapUnloadableWeakConcurrentMap();

        KeyEqualToWeakRefOfItself key = new KeyEqualToWeakRefOfItself();
        Object value = new Object();
        map.put(key, value);
        assertThat(map.containsKey(key), is(true));
        assertThat(map.get(key), is(value));
        assertThat(map.putIfAbsent(key, new Object()), is(value));
        assertThat(map.remove(key), is(value));
        assertThat(map.containsKey(key), is(false));
    }

    private static class MapTestCase {

        private final WeakConcurrentMap<Object, Object> map;

        MapTestCase(WeakConcurrentMap<Object, Object> map) {
            this.map = map;
        }

        void doTest() throws Exception {
            Object
                    key1 = new Object(), value1 = new Object(),
                    key2 = new Object(), value2 = new Object(),
                    key3 = new Object(), value3 = new Object(),
                    key4 = new Object(), value4 = new Object();
            this.map.put(key1, value1);
            this.map.put(key2, value2);
            this.map.put(key3, value3);
            this.map.put(key4, value4);
            assertThat(this.map.get(key1), is(value1));
            assertThat(this.map.get(key2), is(value2));
            assertThat(this.map.get(key3), is(value3));
            assertThat(this.map.get(key4), is(value4));
            Map<Object, Object> values = new HashMap<>();
            values.put(key1, value1);
            values.put(key2, value2);
            values.put(key3, value3);
            values.put(key4, value4);
            for (Map.Entry<Object, Object> entry : this.map) {
                assertThat(values.remove(entry.getKey()), is(entry.getValue()));
            }
            assertThat(values.isEmpty(), is(true));
            key1 = key2 = null; // Make eligible for GC
            System.gc();
            Thread.sleep(200L);
            this.triggerClean();
            assertThat(this.map.get(key3), is(value3));
            assertThat(this.map.getIfPresent(key3), is(value3));
            assertThat(this.map.get(key4), is(value4));
            assertThat(this.map.approximateSize(), is(2));
            assertThat(this.map.target.size(), is(2));
            assertThat(this.map.remove(key3), is(value3));
            assertThat(this.map.get(key3), nullValue());
            assertThat(this.map.getIfPresent(key3), nullValue());
            assertThat(this.map.get(key4), is(value4));
            assertThat(this.map.approximateSize(), is(1));
            assertThat(this.map.target.size(), is(1));
            this.map.clear();
            assertThat(this.map.get(key3), nullValue());
            assertThat(this.map.get(key4), nullValue());
            assertThat(this.map.approximateSize(), is(0));
            assertThat(this.map.target.size(), is(0));
            assertThat(this.map.iterator().hasNext(), is(false));
        }

        protected void triggerClean() {
        }
    }
}