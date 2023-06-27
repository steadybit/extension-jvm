/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.util;

import java.io.File;
import java.io.IOException;
import java.nio.file.FileSystems;
import java.nio.file.Files;
import java.nio.file.attribute.FileAttribute;
import java.nio.file.attribute.PosixFilePermission;
import java.nio.file.attribute.PosixFilePermissions;
import java.util.Set;

public class TempFileUtils {
    private static final boolean isPosix = FileSystems.getDefault().supportedFileAttributeViews().contains("posix");

    @SuppressWarnings("java:S899") // Johannes Edmeier: can't do anything about it
    public static File createTempDir(String prefix) throws IOException {
        if (isPosix) {
            FileAttribute<Set<PosixFilePermission>> attr = PosixFilePermissions.asFileAttribute(PosixFilePermissions.fromString("rwx------"));
            return Files.createTempDirectory(prefix, attr).toFile();
        } else {
            File f = Files.createTempDirectory(prefix).toFile();
            f.setReadable(true, true);
            f.setWritable(true, true);
            f.setExecutable(true, true);
            return f;
        }
    }
}
