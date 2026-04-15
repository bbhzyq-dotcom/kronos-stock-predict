import React, { useState, useEffect } from 'react'

function DataManager({ stocks, onBack }) {
  const [syncStatus, setSyncStatus] = useState(null)
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState(null)

  useEffect(() => {
    fetchSyncStatus()
    const interval = setInterval(fetchSyncStatus, 5000)
    return () => clearInterval(interval)
  }, [])

  const fetchSyncStatus = async () => {
    try {
      const response = await fetch('/api/sync/status')
      if (response.ok) {
        const data = await response.json()
        setSyncStatus(data)
      }
    } catch (err) {
      console.error('Failed to fetch sync status:', err)
    }
  }

  const handleTriggerSync = async () => {
    setLoading(true)
    setMessage(null)
    try {
      const response = await fetch('/api/sync/trigger', { method: 'POST' })
      if (response.ok) {
        setMessage({ type: 'success', text: '同步任务已触发' })
        fetchSyncStatus()
      }
    } catch (err) {
      setMessage({ type: 'error', text: '触发失败' })
    } finally {
      setLoading(false)
    }
  }

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return 'text-green-400'
      case 'running': return 'text-blue-400'
      case 'failed': return 'text-red-400'
      default: return 'text-gray-400'
    }
  }

  const getProgress = () => {
    if (!syncStatus || syncStatus.total_stocks === 0) return 0
    return (syncStatus.processed / syncStatus.total_stocks * 100).toFixed(1)
  }

  return (
    <div>
      <h2 className="text-xl font-bold text-white mb-6">数据管理中心</h2>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-800 rounded-lg p-6">
          <h3 className="text-lg font-medium text-white mb-4">定时同步状态</h3>
          
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-gray-400">当前状态</span>
              <span className={`font-medium ${getStatusColor(syncStatus?.status)}`}>
                {syncStatus?.status || 'idle'}
              </span>
            </div>

            <div className="flex justify-between items-center">
              <span className="text-gray-400">上次同步时间</span>
              <span className="text-white">
                {syncStatus?.last_sync || '-'}
              </span>
            </div>

            <div className="flex justify-between items-center">
              <span className="text-gray-400">股票总数</span>
              <span className="text-white">{syncStatus?.total_stocks || 0}</span>
            </div>

            <div className="flex justify-between items-center">
              <span className="text-gray-400">已处理</span>
              <span className="text-white">{syncStatus?.processed || 0}</span>
            </div>

            {syncStatus?.status === 'running' && (
              <div>
                <div className="flex justify-between items-center mb-2">
                  <span className="text-gray-400">进度</span>
                  <span className="text-white">{getProgress()}%</span>
                </div>
                <div className="bg-slate-700 rounded-full h-3 overflow-hidden">
                  <div
                    className="h-full bg-blue-500 transition-all"
                    style={{ width: `${getProgress()}%` }}
                  ></div>
                </div>
              </div>
            )}
          </div>

          <div className="mt-6 border-t border-slate-700 pt-4">
            <h4 className="text-sm font-medium text-gray-300 mb-3">说明</h4>
            <ul className="text-sm text-gray-500 space-y-1">
              <li>每天下午16:30自动更新K线数据</li>
              <li>自动计算120/30/10/5根K线预测</li>
              <li>预测结果提前算好保存到数据库</li>
              <li>用户打开页面直接查看预测结果</li>
            </ul>
          </div>
        </div>

        <div className="bg-slate-800 rounded-lg p-6">
          <h3 className="text-lg font-medium text-white mb-4">手动同步</h3>
          
          <p className="text-gray-400 text-sm mb-4">
            点击下方按钮可以手动触发同步任务。同步过程会：
          </p>
          <ul className="text-sm text-gray-500 space-y-1 mb-6">
            <li>1. 获取所有A股列表</li>
            <li>2. 获取每只股票近150根日K线</li>
            <li>3. 计算并保存预测结果</li>
          </ul>

          {message && (
            <div className={`mb-4 p-3 rounded ${
              message.type === 'success' ? 'bg-green-900/30 text-green-400' : 'bg-red-900/30 text-red-400'
            }`}>
              {message.text}
            </div>
          )}

          <button
            onClick={handleTriggerSync}
            disabled={loading || syncStatus?.status === 'running'}
            className="w-full px-4 py-3 bg-blue-600 rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? '触发中...' : '手动触发同步'}
          </button>

          <p className="mt-3 text-xs text-gray-500">
            注意：同步过程可能需要较长时间（约10-30分钟），请耐心等待。
          </p>
        </div>
      </div>

      <div className="mt-6 bg-slate-800 rounded-lg p-6">
        <h3 className="text-lg font-medium text-white mb-4">数据统计</h3>
        <div className="grid grid-cols-3 gap-4">
          <div className="text-center">
            <p className="text-2xl font-bold text-white">{stocks.length}</p>
            <p className="text-sm text-gray-400">股票数量</p>
          </div>
          <div className="text-center">
            <p className="text-2xl font-bold text-green-400">
              {stocks.filter(s => s.price > 0).length}
            </p>
            <p className="text-sm text-gray-400">已有价格</p>
          </div>
          <div className="text-center">
            <p className="text-2xl font-bold text-blue-400">
              {stocks.length > 0 ? '150' : '0'}
            </p>
            <p className="text-sm text-gray-400">K线数据</p>
          </div>
        </div>
      </div>
    </div>
  )
}

export default DataManager
