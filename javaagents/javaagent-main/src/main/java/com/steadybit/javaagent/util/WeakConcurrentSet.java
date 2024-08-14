/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.util;

import java.util.Collections;
import java.util.Iterator;
import java.util.Map;

/**
 * <p>
 * A thread-safe set with weak values. Entries are based on a key's system hash code and keys are considered equal only by reference equality.
 * </p>
 * This class does not implement the {@link java.util.Set} interface because this implementation is incompatible
 * with the set contract. While iterating over a set's entries, any value that has not passed iteration is referenced non-weakly.
 */
public class WeakConcurrentSet<V> implements Runnable, Iterable<V> {

    final WeakConcurrentMap<V, Boolean> target;

    public WeakConcurrentSet() {
        this(Cleaner.INLINE);
    }

    public WeakConcurrentSet(Cleaner cleaner) {
        switch (cleaner) {
            case INLINE:
                this.target = new WeakConcurrentMap.WithInlinedExpunction<>();
                break;
            case THREAD:
            case MANUAL:
                this.target = new WeakConcurrentMap<>(cleaner == Cleaner.THREAD);
                break;
            default:
                throw new AssertionError();
        }
    }

    /**
     * @param value The value to add to the set.
     * @return {@code true} if the value was added to the set and was not contained before.
     */
    public boolean add(V value) {
        return this.target.put(value, Boolean.TRUE) == null; // is null or Boolean.TRUE
    }

    /**
     * @param value The value to check if it is contained in the set.
     * @return {@code true} if the set contains the value.
     */
    public boolean contains(V value) {
        return this.target.containsKey(value);
    }

    /**
     * @param value The value to remove from the set.
     * @return {@code true} if the value is contained in the set.
     */
    public boolean remove(V value) {
        return this.target.remove(value) != null; // is null or Boolean.TRUE
    }

    /**
     * Clears the set.
     */
    public void clear() {
        this.target.clear();
    }

    /**
     * Returns the approximate size of this set where the returned number is at least as big as the actual number of entries.
     *
     * @return The minimum size of this set.
     */
    public int approximateSize() {
        return this.target.approximateSize();
    }

    @Override
    public void run() {
        this.target.run();
    }

    /**
     * Determines the cleaning format. A reference is removed either by an explicitly started cleaner thread
     * associated with this instance ({@link Cleaner#THREAD}), as a result of interacting with this thread local
     * from any thread ({@link Cleaner#INLINE} or manually by submitting the detached thread local to a thread
     * ({@link Cleaner#MANUAL}).
     */
    public enum Cleaner {
        THREAD, INLINE, MANUAL
    }

    /**
     * Cleans all unused references.
     */
    public void expungeStaleEntries() {
        this.target.expungeStaleEntries();
    }

    /**
     * @return The cleaner thread or {@code null} if no such thread was set.
     */
    public Thread getCleanerThread() {
        return this.target.getCleanerThread();
    }

    @Override
    public Iterator<V> iterator() {
        return new ReducingIterator<>(this.target.iterator());
    }

    @Override
    public String toString() {
        return Collections.newSetFromMap(this.target.target).toString();
    }

    private static class ReducingIterator<V> implements Iterator<V> {

        private final Iterator<Map.Entry<V, Boolean>> iterator;

        private ReducingIterator(Iterator<Map.Entry<V, Boolean>> iterator) {
            this.iterator = iterator;
        }

        @Override
        public void remove() {
            this.iterator.remove();
        }

        @Override
        public V next() {
            return this.iterator.next().getKey();
        }

        @Override
        public boolean hasNext() {
            return this.iterator.hasNext();
        }
    }
}