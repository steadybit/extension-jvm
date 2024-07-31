/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.util;

import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;

import java.lang.ref.WeakReference;
import java.util.HashMap;
import java.util.Map;

class WeakConcurrentMapTest {

    @Test
    void testLocalExpunction() throws Exception {
        final WeakConcurrentMap.WithInlinedExpunction<Object, Object> map = new WeakConcurrentMap.WithInlinedExpunction<>();
        assertThat(map.getCleanerThread()).isNull();
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
        assertThat(map.getCleanerThread()).isNull();
        Thread thread = new Thread(map);
        thread.start();
        new MapTestCase(map).doTest();
        thread.interrupt();
        Thread.sleep(200L);
        assertThat(thread.isAlive()).isFalse();
    }

    @Test
    void testInternalThread() throws Exception {
        WeakConcurrentMap<Object, Object> map = new WeakConcurrentMap<>(true);
        assertThat(map.getCleanerThread()).isNotNull();
        new MapTestCase(map).doTest();
        map.getCleanerThread().interrupt();
        Thread.sleep(200L);
        assertThat(map.getCleanerThread().isAlive()).isFalse();
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
        assertThat(map.containsKey(key)).isTrue();
        assertThat(map.get(key)).isEqualTo(value);
        assertThat(map.putIfAbsent(key, new Object())).isEqualTo(value);
        assertThat(map.remove(key)).isEqualTo(value);
        assertThat(map.containsKey(key)).isFalse();
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
            assertThat(this.map.get(key1)).isEqualTo(value1);
            assertThat(this.map.get(key2)).isEqualTo(value2);
            assertThat(this.map.get(key3)).isEqualTo(value3);
            assertThat(this.map.get(key4)).isEqualTo(value4);
            Map<Object, Object> values = new HashMap<>();
            values.put(key1, value1);
            values.put(key2, value2);
            values.put(key3, value3);
            values.put(key4, value4);
            for (Map.Entry<Object, Object> entry : this.map) {
                assertThat(values.remove(entry.getKey())).isEqualTo(entry.getValue());
            }
            assertThat(values.isEmpty()).isTrue();
            key1 = key2 = null; // Make eligible for GC
            System.gc();
            Thread.sleep(200L);
            this.triggerClean();
            assertThat(this.map.get(key3)).isEqualTo(value3);
            assertThat(this.map.getIfPresent(key3)).isEqualTo(value3);
            assertThat(this.map.get(key4)).isEqualTo(value4);
            assertThat(this.map.approximateSize()).isEqualTo(2);
            assertThat(this.map.target.size()).isEqualTo(2);
            assertThat(this.map.remove(key3)).isEqualTo(value3);
            assertThat(this.map.get(key3)).isNull();
            assertThat(this.map.getIfPresent(key3)).isNull();
            assertThat(this.map.get(key4)).isEqualTo(value4);
            assertThat(this.map.approximateSize()).isEqualTo(1);
            assertThat(this.map.target.size()).isEqualTo(1);
            this.map.clear();
            assertThat(this.map.get(key3)).isNull();
            assertThat(this.map.get(key4)).isNull();
            assertThat(this.map.approximateSize()).isEqualTo(0);
            assertThat(this.map.target.size()).isEqualTo(0);
            assertThat(this.map.iterator().hasNext()).isFalse();
        }

        protected void triggerClean() {
        }
    }
}
