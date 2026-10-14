package com.haoze.diting.ui

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewModelScope
import com.haoze.diting.data.AppDatabase
import com.haoze.diting.data.ExpressRulesDatabase
import com.haoze.diting.data.RequestSource
import com.haoze.diting.data.RequestStatus
import com.haoze.diting.data.RuleDataset
import com.haoze.diting.data.dao.SubscriptionDao
import com.haoze.diting.data.repository.RequestLogRepository
import androidx.compose.runtime.Immutable
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.util.concurrent.ConcurrentHashMap

/** Records requested per page. */
private const val PAGE_SIZE = 100

@Immutable
data class RequestLogUiState(
    val items: List<RequestLogItem> = emptyList(),
    val source: RequestSource = RequestSource.ALL,
    val status: RequestStatus = RequestStatus.ALL,
    val query: String = "",
    val searching: Boolean = false,
    val loading: Boolean = false,
    val hasMore: Boolean = true,
    /** Total records matching the active filters, resolved for the first page. */
    val total: Int = 0,
    val error: String? = null
)

class RequestLogViewModel @JvmOverloads constructor(
    application: Application,
    initialSource: RequestSource = RequestSource.ALL,
    initialStatus: RequestStatus = RequestStatus.ALL,
    initialQuery: String = "",
    dataset: RuleDataset = RuleDataset.NORMAL
) : AndroidViewModel(application) {
    private val repository = if (dataset == RuleDataset.EXPRESS) {
        RequestLogRepository(
            ExpressRulesDatabase.getInstance(application).dnsLogDao(),
            null
        )
    } else {
        RequestLogRepository(
            AppDatabase.getInstance(application).dnsLogDao(),
            AppDatabase.getInstance(application).httpRequestLogDao()
        )
    }
    private val subscriptionDao: SubscriptionDao = if (dataset == RuleDataset.EXPRESS) {
        ExpressRulesDatabase.getInstance(application).subscriptionDao()
    } else {
        AppDatabase.getInstance(application).subscriptionDao()
    }
    private var subscriptionNames: Map<Long, String> = emptyMap()
    private val _state = MutableStateFlow(
        RequestLogUiState(
            source = initialSource,
            status = initialStatus,
            query = initialQuery,
            searching = initialQuery.isNotEmpty()
        )
    )
    val state: StateFlow<RequestLogUiState> = _state.asStateFlow()
    private var limit = PAGE_SIZE
    private var searchJob: Job? = null
    private var loadJob: Job? = null
    private val appNameCache = ConcurrentHashMap<String, String>()

    /** Resolves the app label for a package, cached because pages are rebuilt on every load. */
    private fun resolveAppName(packageName: String?): String? {
        if (packageName.isNullOrBlank()) return null
        appNameCache[packageName]?.let { return it }
        val label = runCatching {
            val manager = getApplication<Application>().packageManager
            @Suppress("DEPRECATION")
            val info = manager.getApplicationInfo(packageName, 0)
            manager.getApplicationLabel(info).toString()
        }.getOrNull()?.takeIf { it.isNotBlank() }
        if (label != null) appNameCache[packageName] = label
        return label
    }

    /** Resolved once per load; the subscription table is tiny and changes rarely. */
    private suspend fun refreshSubscriptionNames() {
        subscriptionNames = try {
            subscriptionDao.all().associate { it.id to it.name }
        } catch (e: CancellationException) {
            throw e
        } catch (_: Exception) {
            emptyMap()
        }
    }

    init { refresh() }

    fun refresh() {
        limit = PAGE_SIZE
        load()
    }

    fun loadMore() {
        if (_state.value.loading || !_state.value.hasMore) return
        limit += PAGE_SIZE
        load()
    }

    fun setSource(source: RequestSource) {
        if (_state.value.source == source) return
        _state.value = _state.value.copy(source = source)
        refresh()
    }

    fun setStatus(status: RequestStatus) {
        if (_state.value.status == status) return
        _state.value = _state.value.copy(status = status)
        refresh()
    }

    fun setQuery(query: String) {
        _state.value = _state.value.copy(query = query)
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            if (query.isNotBlank()) delay(250)
            limit = PAGE_SIZE
            load()
        }
    }

    fun setSearching(value: Boolean) {
        _state.value = _state.value.copy(searching = value)
        if (!value && _state.value.query.isNotEmpty()) {
            setQuery("")
        }
    }

    private fun load() {
        loadJob?.cancel()
        val currentSource = _state.value.source
        val currentStatus = _state.value.status
        val currentQuery = _state.value.query
        val currentLimit = limit

        loadJob = viewModelScope.launch(Dispatchers.IO) {
            _state.value = _state.value.copy(loading = true, error = null)
            runCatching {
                val batch = repository.load(
                    limit = currentLimit,
                    source = currentSource,
                    status = currentStatus,
                    query = currentQuery
                )
                // Resolved for the first page only; later pages reuse it so that loading more
                // does not re-run an expensive COUNT on every scroll.
                val total = if (currentLimit <= PAGE_SIZE) {
                    repository.count(
                        source = currentSource,
                        status = currentStatus,
                        query = currentQuery
                    )
                } else {
                    _state.value.total
                }
                batch to total
            }.onSuccess { (batch, total) ->
                // Subscription names change rarely: resolve them on the first page only, so
                // "load more" does not re-read the subscription table every time.
                if (currentLimit <= PAGE_SIZE) {
                    refreshSubscriptionNames()
                }
                val allItems = (batch.dns.map(::dnsRequestItem) + batch.http.map(::httpRequestItem))
                    .map { item ->
                        item.copy(
                            appName = resolveAppName(item.packageName),
                            subscriptionName = item.blockSubscriptionId?.let { subscriptionNames[it] }
                        )
                    }
                    .sortedByDescending { it.timestamp }
                val hasMore = batch.hasMore || allItems.size > currentLimit
                val items = allItems.take(currentLimit)
                _state.value = _state.value.copy(
                    items = items,
                    loading = false,
                    hasMore = hasMore,
                    total = total
                )
            }.onFailure { error ->
                _state.value = _state.value.copy(
                    loading = false,
                    error = error.message ?: "加载失败"
                )
            }
        }
    }

    companion object {
        fun factory(
            application: Application,
            initialSource: RequestSource = RequestSource.ALL,
            initialStatus: RequestStatus = RequestStatus.ALL,
            initialQuery: String = "",
            dataset: RuleDataset = RuleDataset.NORMAL
        ): ViewModelProvider.Factory = object : ViewModelProvider.Factory {
            @Suppress("UNCHECKED_CAST")
            override fun <T : ViewModel> create(modelClass: Class<T>): T {
                return RequestLogViewModel(
                    application,
                    initialSource,
                    initialStatus,
                    initialQuery,
                    dataset
                ) as T
            }
        }
    }
}
