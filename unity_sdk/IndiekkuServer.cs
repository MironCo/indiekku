using System;
using System.Net;
using System.Text;
using System.Threading;
using UnityEngine;

/// <summary>
/// Drop this component onto a persistent GameObject in your server scene.
/// It starts a tiny HTTP server on port 9999 that indiekku polls to track
/// player count and server capacity — no API key or outbound calls needed.
///
/// Usage:
///   IndiekkuServer.Instance.SetPlayerCount(NetworkManager.Singleton.ConnectedClients.Count);
/// </summary>
public class IndiekkuServer : MonoBehaviour
{
    public static IndiekkuServer Instance { get; private set; }

    [Tooltip("Maximum number of players this server accepts.")]
    [SerializeField] private int maxPlayers = 8;

    [Tooltip("Internal port indiekku polls for status. Must match the value indiekku was built with (default 9999).")]
    [SerializeField] private int statusPort = 9999;

    private int _playerCount = 0;
    private HttpListener _listener;
    private Thread _listenerThread;
    private volatile bool _running = false;

    private void Awake()
    {
        if (Instance != null && Instance != this)
        {
            Destroy(gameObject);
            return;
        }
        Instance = this;
        DontDestroyOnLoad(gameObject);
    }

    private void Start()
    {
        StartStatusServer();
    }

    /// <summary>
    /// Call this whenever your player count changes, e.g. OnClientConnected / OnClientDisconnected.
    /// Thread-safe.
    /// </summary>
    public void SetPlayerCount(int count)
    {
        Interlocked.Exchange(ref _playerCount, count);
    }

    /// <summary>
    /// Override max players at runtime (e.g. for different game modes).
    /// </summary>
    public void SetMaxPlayers(int max)
    {
        maxPlayers = max;
    }

    private void StartStatusServer()
    {
        _listener = new HttpListener();
        _listener.Prefixes.Add($"http://*:{statusPort}/");

        try
        {
            _listener.Start();
        }
        catch (Exception e)
        {
            Debug.LogError($"[IndiekkuServer] Failed to start status server on port {statusPort}: {e.Message}");
            return;
        }

        _running = true;
        _listenerThread = new Thread(ListenLoop) { IsBackground = true, Name = "IndiekkuStatusServer" };
        _listenerThread.Start();
        Debug.Log($"[IndiekkuServer] Status server listening on :{statusPort}");
    }

    private void ListenLoop()
    {
        while (_running && _listener.IsListening)
        {
            HttpListenerContext ctx;
            try
            {
                ctx = _listener.GetContext();
            }
            catch (HttpListenerException)
            {
                break; // listener was stopped
            }
            catch (ObjectDisposedException)
            {
                break;
            }

            try
            {
                var req = ctx.Request;
                var res = ctx.Response;

                if (req.HttpMethod == "GET" && req.Url.AbsolutePath == "/status")
                {
                    int pc = Interlocked.CompareExchange(ref _playerCount, 0, -1); // atomic read
                    var json = $"{{\"player_count\":{pc},\"max_players\":{maxPlayers}}}";
                    var bytes = Encoding.UTF8.GetBytes(json);
                    res.StatusCode = 200;
                    res.ContentType = "application/json";
                    res.ContentLength64 = bytes.Length;
                    res.OutputStream.Write(bytes, 0, bytes.Length);
                }
                else
                {
                    res.StatusCode = 404;
                }

                res.Close();
            }
            catch (Exception e)
            {
                Debug.LogWarning($"[IndiekkuServer] Error handling request: {e.Message}");
                try { ctx.Response.Abort(); } catch { }
            }
        }
    }

    private void OnDestroy()
    {
        _running = false;
        try { _listener?.Stop(); } catch { }
        _listenerThread?.Join(500);
    }
}
