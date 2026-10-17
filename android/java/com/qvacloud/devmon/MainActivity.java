package com.qvacloud.devmon;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Context;
import android.content.SharedPreferences;
import android.graphics.Color;
import android.graphics.Typeface;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.text.InputType;
import android.util.TypedValue;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.Window;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.widget.EditText;
import android.widget.HorizontalScrollView;
import android.widget.LinearLayout;
import android.widget.TextView;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class MainActivity extends Activity {

    // AMOLED palette + terminal green
    static final int BG    = 0xFF000000;
    static final int GREEN = 0xFF39FF14;
    static final int MID   = 0xFF1F8A1F;
    static final int DIM   = 0xFF0E3D0E;
    static final int WARN  = 0xFFD7FF00;
    static final int BAD   = 0xFFFF2B2B;

    static final int POLL_MS = 5000;   // low frequency -> low power
    static final int TIMEOUT = 1800;   // ms connect/read

    static final Typeface MONO = Typeface.MONOSPACE;

    final Handler ui = new Handler(Looper.getMainLooper());
    ExecutorService pool;
    final List<Endpoint> eps = new ArrayList<>();
    final List<DevCard> cards = new ArrayList<>();
    SharedPreferences prefs;
    LinearLayout list;
    TextView status;
    volatile boolean running = false;

    static class Endpoint {
        final String name, url;
        Endpoint(String n, String u) { name = n; url = u; }
    }

    @Override protected void onCreate(Bundle b) {
        super.onCreate(b);
        prefs = getSharedPreferences("devmon", MODE_PRIVATE);
        buildUi();
        loadEndpoints();
        rebuild();
    }

    @Override protected void onStart() {
        super.onStart();
        running = true;
        pool = Executors.newFixedThreadPool(4);
        ui.postDelayed(loop, 300);
    }

    @Override protected void onResume() { super.onResume(); immersive(); }

    @Override public void onWindowFocusChanged(boolean hasFocus) {
        super.onWindowFocusChanged(hasFocus);
        if (hasFocus) immersive();
    }

    @Override protected void onStop() {
        running = false;
        ui.removeCallbacksAndMessages(null);
        if (pool != null) pool.shutdownNow();
        super.onStop();
    }

    final Runnable loop = new Runnable() {
        @Override public void run() {
            if (!running) return;
            poll();
            ui.postDelayed(this, POLL_MS);
        }
    };

    private void immersive() {
        Window w = getWindow();
        w.setStatusBarColor(Color.TRANSPARENT);
        w.setNavigationBarColor(Color.TRANSPARENT);
        if (Build.VERSION.SDK_INT >= 30) {
            w.setDecorFitsSystemWindows(false);
            WindowInsetsController c = w.getInsetsController();
            if (c != null) {
                c.hide(WindowInsets.Type.systemBars());
                c.setSystemBarsBehavior(WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE);
            }
        } else {
            w.getDecorView().setSystemUiVisibility(
                View.SYSTEM_UI_FLAG_FULLSCREEN | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
                | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY | View.SYSTEM_UI_FLAG_LAYOUT_STABLE);
        }
    }

    private int dp(float v) {
        return (int) (v * getResources().getDisplayMetrics().density + 0.5f);
    }

    private void buildUi() {
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setBackgroundColor(BG);
        root.setPadding(dp(10), dp(14), dp(10), dp(6));

        LinearLayout head = new LinearLayout(this);
        head.setOrientation(LinearLayout.HORIZONTAL);
        head.setGravity(Gravity.CENTER_VERTICAL);
        head.setPadding(0, dp(2), 0, dp(8));

        TextView title = new TextView(this);
        title.setText("\u25C8 DEVMON");
        title.setTextColor(GREEN);
        title.setTypeface(MONO, Typeface.BOLD);
        title.setTextSize(TypedValue.COMPLEX_UNIT_SP, 15);
        head.addView(title);

        status = new TextView(this);
        status.setTextColor(MID);
        status.setTypeface(MONO);
        status.setTextSize(TypedValue.COMPLEX_UNIT_SP, 10);
        LinearLayout.LayoutParams sp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
        sp.leftMargin = dp(10);
        status.setLayoutParams(sp);
        head.addView(status);

        TextView add = new TextView(this);
        add.setText("+ ADD");
        add.setTextColor(GREEN);
        add.setTypeface(MONO, Typeface.BOLD);
        add.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        add.setPadding(dp(12), dp(6), dp(6), dp(6));
        add.setOnClickListener(v -> addDialog());
        head.addView(add);

        root.addView(head);

        View line = new View(this);
        line.setBackgroundColor(DIM);
        line.setLayoutParams(new LinearLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, dp(1)));
        root.addView(line);

        HorizontalScrollView scroll = new HorizontalScrollView(this);
        scroll.setFillViewport(true);
        scroll.setOverScrollMode(View.OVER_SCROLL_NEVER);
        scroll.setHorizontalScrollBarEnabled(false);
        list = new LinearLayout(this);
        list.setOrientation(LinearLayout.HORIZONTAL);
        list.setPadding(0, dp(6), 0, dp(10));
        scroll.addView(list, new HorizontalScrollView.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.MATCH_PARENT));
        root.addView(scroll, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));

        setContentView(root);
    }

    // ---- endpoints persistence ----
    private void loadEndpoints() {
        eps.clear();
        String raw = prefs.getString("endpoints", null);
        if (raw == null) {
            eps.add(new Endpoint("PC", "http://192.168.1.10:55555"));
            eps.add(new Endpoint("DEVICE-2", "http://192.168.1.11:55555"));
            eps.add(new Endpoint("DEVICE-3", "http://192.168.1.12:55555"));
            saveEndpoints();
            return;
        }
        try {
            JSONArray a = new JSONArray(raw);
            for (int i = 0; i < a.length(); i++) {
                JSONObject o = a.getJSONObject(i);
                eps.add(new Endpoint(o.optString("name"), o.optString("url")));
            }
        } catch (Exception ignored) {}
    }

    private void saveEndpoints() {
        JSONArray a = new JSONArray();
        for (Endpoint e : eps) {
            JSONObject o = new JSONObject();
            try { o.put("name", e.name); o.put("url", e.url); } catch (Exception ignored) {}
            a.put(o);
        }
        prefs.edit().putString("endpoints", a.toString()).apply();
    }

    private void rebuild() {
        list.removeAllViews();
        cards.clear();
        int n = Math.max(1, eps.size());
        int avail = getResources().getDisplayMetrics().widthPixels - dp(20);
        int cardW = Math.max(dp(230), avail / n);
        for (int i = 0; i < eps.size(); i++) {
            DevCard c = new DevCard(this, eps.get(i), i);
            cards.add(c);
            list.addView(c, new LinearLayout.LayoutParams(cardW, ViewGroup.LayoutParams.MATCH_PARENT));
            if (i < eps.size() - 1) {
                View sep = new View(this);
                LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                        dp(1), ViewGroup.LayoutParams.MATCH_PARENT);
                lp.setMargins(dp(10), dp(6), dp(10), dp(6));
                sep.setLayoutParams(lp);
                sep.setBackgroundColor(DIM);
                list.addView(sep);
            }
        }
        status.setText(eps.size() + (eps.size() == 1 ? " host" : " hosts"));
    }

    private void addDialog() {
        LinearLayout box = new LinearLayout(this);
        box.setOrientation(LinearLayout.VERTICAL);
        box.setPadding(dp(18), dp(8), dp(18), 0);
        final EditText name = new EditText(this);
        name.setHint("Nombre (p.ej. OrangePi-3)");
        name.setInputType(InputType.TYPE_CLASS_TEXT);
        final EditText url = new EditText(this);
        url.setHint("http://192.168.1.x:55555");
        url.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        box.addView(name);
        box.addView(url);

        new AlertDialog.Builder(this)
            .setTitle("A\u00f1adir endpoint")
            .setView(box)
            .setPositiveButton("Guardar", (d, w) -> {
                String u = url.getText().toString().trim();
                if (u.isEmpty()) return;
                if (!u.startsWith("http")) u = "http://" + u;
                String n = name.getText().toString().trim();
                if (n.isEmpty()) n = u;
                eps.add(new Endpoint(n, u));
                saveEndpoints();
                rebuild();
                ui.postDelayed(loop, 50);
            })
            .setNegativeButton("Cancelar", null)
            .show();
    }

    private void deleteEndpoint(int idx) {
        if (idx < 0 || idx >= eps.size()) return;
        Endpoint e = eps.get(idx);
        new AlertDialog.Builder(this)
            .setTitle("Eliminar " + e.name + "?")
            .setPositiveButton("Eliminar", (d, w) -> {
                eps.remove(idx);
                saveEndpoints();
                rebuild();
            })
            .setNegativeButton("Cancelar", null)
            .show();
    }

    // ---- polling ----
    private void poll() {
        for (int i = 0; i < eps.size(); i++) {
            final int idx = i;
            final Endpoint e = eps.get(i);
            pool.submit(() -> {
                long t0 = System.currentTimeMillis();
                try {
                    JSONObject m = fetch(e.url);
                    long ms = System.currentTimeMillis() - t0;
                    ui.post(() -> {
                        if (idx < cards.size()) cards.get(idx).update(m, ms);
                    });
                } catch (Exception ex) {
                    ui.post(() -> {
                        if (idx < cards.size()) cards.get(idx).offline(shortMsg(ex));
                    });
                }
            });
        }
    }

    private JSONObject fetch(String base) throws Exception {
        String url = base.endsWith("/metrics") ? base : base + "/metrics";
        HttpURLConnection c = (HttpURLConnection) new URL(url).openConnection();
        try {
            c.setConnectTimeout(TIMEOUT);
            c.setReadTimeout(TIMEOUT);
            c.setRequestProperty("Accept", "application/json");
            c.setRequestProperty("Connection", "close");
            int code = c.getResponseCode();
            if (code != 200) throw new Exception("HTTP " + code);
            StringBuilder sb = new StringBuilder();
            try (BufferedReader r = new BufferedReader(new InputStreamReader(c.getInputStream()))) {
                String ln;
                while ((ln = r.readLine()) != null) sb.append(ln);
            }
            return new JSONObject(sb.toString());
        } finally {
            c.disconnect();
        }
    }

    private String shortMsg(Exception e) {
        String m = e.getMessage();
        if (m == null) m = e.getClass().getSimpleName();
        return m.length() > 40 ? m.substring(0, 40) : m;
    }

    static String fmtBytes(double b) {
        if (b < 0) return "\u2014";
        String[] u = {"B", "K", "M", "G", "T", "P"};
        int i = 0;
        double v = b;
        while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
        return (v >= 100 || i == 0 ? String.format("%.0f", v) : String.format("%.1f", v)) + u[i];
    }

    static String fmtUptime(double s) {
        if (s <= 0) return "\u2014";
        int d = (int) (s / 86400), h = (int) ((s % 86400) / 3600), m = (int) ((s % 3600) / 60);
        if (d > 0) return d + "d " + h + "h";
        if (h > 0) return h + "h " + m + "m";
        return m + "m";
    }

    static int levelColor(double p) {
        if (p >= 90) return BAD;
        if (p >= 75) return WARN;
        return GREEN;
    }

    // ---------- one device division ----------
    class DevCard extends LinearLayout {
        final Endpoint ep;
        final int index;
        TextView dot, name, stat;
        StatRow cpu, ram, swap, tmp;
        LinearLayout diskBox;

        DevCard(Context ctx, Endpoint e, int idx) {
            super(ctx);
            ep = e; index = idx;
            setOrientation(VERTICAL);
            setPadding(dp(4), dp(10), dp(4), dp(12));
            setOnLongClickListener(v -> { deleteEndpoint(index); return true; });

            LinearLayout h = new LinearLayout(ctx);
            h.setOrientation(HORIZONTAL);
            h.setGravity(Gravity.CENTER_VERTICAL);

            dot = new TextView(ctx);
            dot.setText("\u25CF");
            dot.setTextColor(MID);
            dot.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
            dot.setPadding(0, 0, dp(6), 0);
            h.addView(dot);

            name = new TextView(ctx);
            name.setText(ep.name.toUpperCase());
            name.setTextColor(GREEN);
            name.setTypeface(MONO, Typeface.BOLD);
            name.setTextSize(TypedValue.COMPLEX_UNIT_SP, 16);
            h.addView(name);

            stat = new TextView(ctx);
            stat.setText("\u00b7\u00b7\u00b7");
            stat.setTextColor(MID);
            stat.setTypeface(MONO);
            stat.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
            LinearLayout.LayoutParams sp = new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f);
            stat.setLayoutParams(sp);
            stat.setGravity(Gravity.END);
            h.addView(stat);

            addView(h);

            TextView sub = new TextView(ctx);
            sub.setText(ep.url);
            sub.setTextColor(MID);
            sub.setTypeface(MONO);
            sub.setTextSize(TypedValue.COMPLEX_UNIT_SP, 11);
            sub.setPadding(dp(16), dp(1), 0, dp(2));
            addView(sub);

            cpu = new StatRow(ctx);
            ram = new StatRow(ctx);
            swap = new StatRow(ctx);
            tmp = new StatRow(ctx);
            addWeighted(cpu, 1);
            addWeighted(ram, 1);
            addWeighted(swap, 1);
            addWeighted(tmp, 1);

            diskBox = new LinearLayout(ctx);
            diskBox.setOrientation(VERTICAL);
            addWeighted(diskBox, 1);
        }

        void addWeighted(View v, float w) {
            addView(v, new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, 0, w));
        }

        void setBar(StatRow r, String label, String value, double pct) {
            r.label.setText(label);
            r.value.setText(value);
            r.setFill(pct, levelColor(pct));
        }

        void update(JSONObject m, long ms) {
            try {
                dot.setTextColor(GREEN);
                stat.setText("ONLINE " + ms + "ms");
                stat.setTextColor(MID);

                JSONObject cpuJ = m.optJSONObject("cpu");
                JSONObject memJ = m.optJSONObject("mem");
                JSONObject swpJ = m.optJSONObject("swap");

                double cu = cpuJ != null ? cpuJ.optDouble("usage_percent", 0) : 0;
                int cores = cpuJ != null ? cpuJ.optInt("cores", 0) : 0;
                double load = cpuJ != null ? cpuJ.optDouble("load1", 0) : 0;
                String os = m.optString("os"), arch = m.optString("arch");
                setBar(cpu, "CPU", String.format("%.1f%%", cu) + "  " + cores + "c L" + String.format("%.2f", load), cu);

                double mp = memJ != null ? memJ.optDouble("used_percent", 0) : 0;
                double mu = memJ != null ? memJ.optDouble("used", 0) : 0;
                double mt = memJ != null ? memJ.optDouble("total", 0) : 0;
                setBar(ram, "RAM", fmtBytes(mu) + "/" + fmtBytes(mt) + " " + String.format("%.0f%%", mp), mp);

                double sp2 = swpJ != null ? swpJ.optDouble("used_percent", 0) : 0;
                double su = swpJ != null ? swpJ.optDouble("used", 0) : 0;
                double st = swpJ != null ? swpJ.optDouble("total", 0) : 0;
                if (st > 0) {
                    swap.setVisibility(VISIBLE);
                    setBar(swap, "SWAP", fmtBytes(su) + "/" + fmtBytes(st) + " " + String.format("%.0f%%", sp2), sp2);
                } else {
                    swap.setVisibility(GONE);
                }

                JSONArray temps = m.optJSONArray("temps");
                StringBuilder tb = new StringBuilder();
                double hottest = 0;
                if (temps != null) {
                    for (int i = 0; i < temps.length(); i++) {
                        JSONObject t = temps.getJSONObject(i);
                        double c = t.optDouble("celsius", 0);
                        if (c > hottest) hottest = c;
                        if (tb.length() > 0) tb.append("   ");
                        tb.append(shortLabel(t.optString("label"))).append(' ')
                          .append(String.format("%.0f", c)).append('\u00b0');
                    }
                }
                tmp.label.setText("TMP");
                tmp.value.setText(tb.length() == 0 ? "sin sensores" : tb.toString());
                tmp.value.setTextColor(tb.length() == 0 ? MID : (hottest >= 80 ? BAD : hottest >= 65 ? WARN : GREEN));
                tmp.hideBar();

                diskBox.removeAllViews();
                JSONArray disks = m.optJSONArray("disks");
                int nd = disks != null ? disks.length() : 0;
                LinearLayout.LayoutParams dp_lp = (LinearLayout.LayoutParams) diskBox.getLayoutParams();
                dp_lp.weight = Math.max(1, nd);
                diskBox.setLayoutParams(dp_lp);
                if (disks != null) {
                    for (int i = 0; i < disks.length(); i++) {
                        JSONObject d = disks.getJSONObject(i);
                        double p = d.optDouble("used_percent", 0);
                        StatRow dr = new StatRow(getContext());
                        String mount = d.optString("mount");
                        String fs = d.optString("fs");
                        setBar(dr, "DSK " + mount + (fs.isEmpty() ? "" : " " + fs),
                                fmtBytes(d.optDouble("used", 0)) + "/" + fmtBytes(d.optDouble("total", 0))
                                + " " + String.format("%.0f%%", p), p);
                        diskBox.addView(dr, new LinearLayout.LayoutParams(
                                ViewGroup.LayoutParams.MATCH_PARENT, 0, 1f));
                    }
                }

                name.setTextColor(GREEN);
            } catch (Exception e) {
                offline(shortMsg(e));
            }
        }

        void offline(String why) {
            dot.setTextColor(BAD);
            stat.setText("OFFLINE");
            stat.setTextColor(BAD);
            name.setTextColor(BAD);
            cpu.value.setText(why);
            cpu.setFill(100, BAD);
            ram.value.setText("\u2014"); ram.setFill(0, MID);
            swap.setVisibility(GONE);
            tmp.value.setText("\u2014"); tmp.hideBar();
            diskBox.removeAllViews();
        }

        String shortLabel(String l) {
            l = l.replace("_zone", "").replace("thermal_", "").replace("_thermal", "");
            return l.length() > 12 ? l.substring(0, 12) : l;
        }
    }

    // ---------- thin bar row ----------
    class StatRow extends LinearLayout {
        TextView label, value;
        View fill, spacer;
        LinearLayout track;

        StatRow(Context ctx) {
            super(ctx);
            setOrientation(VERTICAL);
            setGravity(Gravity.CENTER_VERTICAL);
            setPadding(dp(2), dp(4), dp(2), dp(4));
            LinearLayout top = new LinearLayout(ctx);
            top.setOrientation(HORIZONTAL);
            label = new TextView(ctx);
            label.setTextColor(MID);
            label.setTypeface(MONO);
            label.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13);
            value = new TextView(ctx);
            value.setTextColor(GREEN);
            value.setTypeface(MONO);
            value.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13);
            value.setGravity(Gravity.END);
            top.addView(label, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f));
            top.addView(value);
            addView(top);

            track = new LinearLayout(ctx);
            track.setOrientation(HORIZONTAL);
            track.setBackgroundColor(DIM);
            fill = new View(ctx);
            fill.setBackgroundColor(GREEN);
            spacer = new View(ctx);
            track.addView(fill, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.MATCH_PARENT, 1f));
            track.addView(spacer, new LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.MATCH_PARENT, 99f));
            LinearLayout.LayoutParams tlp = new LinearLayout.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT, dp(5));
            tlp.topMargin = dp(5);
            addView(track, tlp);
        }

        void setFill(double pct, int color) {
            float f = (float) Math.max(0.01, Math.min(100, pct));
            float r = (float) Math.max(0.01, 100 - f);
            LinearLayout.LayoutParams a = (LinearLayout.LayoutParams) fill.getLayoutParams();
            a.weight = f; fill.setLayoutParams(a);
            LinearLayout.LayoutParams b = (LinearLayout.LayoutParams) spacer.getLayoutParams();
            b.weight = r; spacer.setLayoutParams(b);
            fill.setBackgroundColor(color);
        }

        void hideBar() { track.setVisibility(GONE); }
    }
}
