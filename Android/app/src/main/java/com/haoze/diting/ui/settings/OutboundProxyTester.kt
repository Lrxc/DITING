package com.haoze.diting.ui.settings

import android.util.Base64
import com.haoze.diting.ui.OutboundProxyProtocol
import java.io.BufferedInputStream
import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.InputStream
import java.io.OutputStream
import java.net.InetSocketAddress
import java.net.Socket
import java.nio.charset.StandardCharsets

/**
 * Probes whether the configured local proxy is reachable right now.
 *
 * The probe runs on plain sockets, so it works regardless of the outbound
 * proxy switch and even while the VPN is stopped. SOCKS5 and HTTP CONNECT are
 * exercised with a real handshake (plus authentication when credentials are
 * given) so an open port alone is not reported as available.
 */
object OutboundProxyTester {
    private const val TIMEOUT_MS = 4000
    private const val PROBE_HOST = "www.baidu.com"
    private const val PROBE_PORT = 443

    sealed interface Result {
        data object Available : Result
        data class Unavailable(val reason: String) : Result
    }

    fun test(
        protocol: OutboundProxyProtocol,
        host: String,
        port: Int,
        username: String,
        password: String
    ): Result = runCatching {
        Socket().use { socket ->
            socket.connect(InetSocketAddress(host, port), TIMEOUT_MS)
            socket.soTimeout = TIMEOUT_MS
            val output = socket.getOutputStream()
            val input = BufferedInputStream(socket.getInputStream())
            when (protocol) {
                OutboundProxyProtocol.SOCKS5 -> probeSocks5(output, input, username, password)
                OutboundProxyProtocol.HTTP -> probeHttpConnect(output, input, username, password)
            }
        }
        Result.Available
    }.getOrElse { error ->
        Result.Unavailable(describe(error))
    }

    private fun probeSocks5(output: OutputStream, input: InputStream, username: String, password: String) {
        val hasCredentials = username.isNotEmpty() || password.isNotEmpty()
        output.write(
            if (hasCredentials) byteArrayOf(0x05, 0x02, 0x00, 0x02)
            else byteArrayOf(0x05, 0x01, 0x00)
        )
        output.flush()
        val greeting = readFully(input, 2)
        if (greeting[0].toInt() and 0xFF != 0x05) throw IOException("代理返回异常响应")
        when (greeting[1].toInt() and 0xFF) {
            0x00 -> Unit
            0x02 -> socks5Authenticate(output, input, username, password)
            0xFF -> throw IOException("代理不接受当前的认证方式")
            else -> throw IOException("代理不支持无认证连接")
        }
        val hostBytes = PROBE_HOST.toByteArray(StandardCharsets.US_ASCII)
        val request = ByteArrayOutputStream()
        request.write(byteArrayOf(0x05, 0x01, 0x00, 0x03, hostBytes.size.toByte()))
        request.write(hostBytes)
        request.write(byteArrayOf((PROBE_PORT shr 8).toByte(), (PROBE_PORT and 0xFF).toByte()))
        output.write(request.toByteArray())
        output.flush()

        val reply = readFully(input, 4)
        if (reply[0].toInt() and 0xFF != 0x05) throw IOException("代理返回异常响应")
        when (reply[1].toInt() and 0xFF) {
            0x00 -> Unit
            0x02 -> throw IOException("代理拒绝连接目标")
            0x05 -> throw IOException("代理拒绝连接目标")
            else -> throw IOException("代理连接目标失败")
        }
        drainSocks5Address(input, reply[3].toInt() and 0xFF)
    }

    private fun socks5Authenticate(output: OutputStream, input: InputStream, username: String, password: String) {
        val user = username.toByteArray(StandardCharsets.UTF_8)
        val pass = password.toByteArray(StandardCharsets.UTF_8)
        output.write(byteArrayOf(0x01, user.size.toByte()))
        output.write(user)
        output.write(pass.size)
        output.write(pass)
        output.flush()
        val reply = readFully(input, 2)
        if (reply[0].toInt() and 0xFF != 0x01 || reply[1].toInt() and 0xFF != 0x00) {
            throw IOException("代理认证失败")
        }
    }

    private fun drainSocks5Address(input: InputStream, addressType: Int) {
        when (addressType) {
            0x01 -> readFully(input, 4 + 2)
            0x03 -> {
                val length = readFully(input, 1)[0].toInt() and 0xFF
                readFully(input, length + 2)
            }
            0x04 -> readFully(input, 16 + 2)
        }
    }

    private fun probeHttpConnect(output: OutputStream, input: InputStream, username: String, password: String) {
        val target = "$PROBE_HOST:$PROBE_PORT"
        val request = buildString {
            append("CONNECT ").append(target).append(" HTTP/1.1\r\n")
            append("Host: ").append(target).append("\r\n")
            if (username.isNotEmpty() || password.isNotEmpty()) {
                val token = Base64.encodeToString(
                    "$username:$password".toByteArray(StandardCharsets.UTF_8),
                    Base64.NO_WRAP
                )
                append("Proxy-Authorization: Basic ").append(token).append("\r\n")
            }
            append("\r\n")
        }
        output.write(request.toByteArray(StandardCharsets.US_ASCII))
        output.flush()

        val statusLine = readLine(input)
        if (statusLine.isBlank()) throw IOException("代理返回异常响应")
        val status = statusLine.split(' ').getOrNull(1)?.toIntOrNull()
            ?: throw IOException("代理返回异常响应")
        when {
            status == 407 -> throw IOException("代理认证失败")
            status in 200..299 -> Unit
            else -> throw IOException("代理返回 HTTP $status")
        }
    }

    private fun readLine(input: InputStream): String {
        val builder = StringBuilder()
        while (builder.length < 256) {
            val byte = input.read()
            if (byte < 0 || byte == '\n'.code) break
            if (byte != '\r'.code) builder.append(byte.toChar())
        }
        return builder.toString()
    }

    private fun readFully(input: InputStream, count: Int): ByteArray {
        val buffer = ByteArray(count)
        var offset = 0
        while (offset < count) {
            val read = input.read(buffer, offset, count - offset)
            if (read < 0) throw IOException("代理提前断开连接")
            offset += read
        }
        return buffer
    }

    private fun describe(error: Throwable): String {
        val message = error.message.orEmpty()
        val lower = message.lowercase()
        return when {
            message.startsWith("代理") -> message
            lower.contains("refused") -> "代理端口未开放或代理未启动"
            lower.contains("timed out") || lower.contains("timeout") -> "连接代理超时"
            lower.contains("unreachable") -> "代理地址不可达"
            lower.contains("permission") -> "无权访问代理端口"
            else -> message.ifBlank { "无法连接代理" }
        }
    }
}
